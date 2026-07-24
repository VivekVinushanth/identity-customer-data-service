/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

// Package worker consumes events off the orchestration queue and, for each
// one, matches active orchestration rules, evaluates their conditions
// against the event and its profile, and runs the matched actions. It is
// deliberately a separate package from internal/system/workers: action
// executors (internal/orchestration/executor) call back into
// internal/system/workers (e.g. EnqueueProfileForProcessing for the
// profile.merge action) — keeping the consumer loop here avoids an import
// cycle between the two.
package worker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	eventModel "github.com/wso2/identity-customer-data-service/internal/event/model"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/executor"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/matcher"
	orchestrationModel "github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	orchestrationStore "github.com/wso2/identity-customer-data-service/internal/orchestration/store"
	profileProvider "github.com/wso2/identity-customer-data-service/internal/profile/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/config"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
	"github.com/wso2/identity-customer-data-service/internal/system/queue"
)

// activeQueue is the queue implementation used for orchestration event
// processing. It is initialised by StartOrchestrationWorker. All access is
// guarded by queueMu to prevent data races between concurrent Enqueue calls
// and shutdown.
var (
	queueMu     sync.RWMutex
	activeQueue queue.OrchestrationQueue
)

// StartOrchestrationWorker initialises the orchestration queue (using the
// provider configured in the runtime config) and starts the consumer
// goroutine. An error is returned when the queue cannot be created or
// started; the caller should treat this as a fatal startup failure.
func StartOrchestrationWorker() error {
	cfg := config.GetCDSRuntime().Config
	q, err := queue.NewOrchestrationQueue(cfg)
	if err != nil {
		return fmt.Errorf("worker: failed to create orchestration queue: %w", err)
	}
	if err := q.Start(processEvent); err != nil {
		_ = q.Close()
		return fmt.Errorf("worker: failed to start orchestration queue: %w", err)
	}
	queueMu.Lock()
	activeQueue = q
	queueMu.Unlock()
	return nil
}

// EnqueueEventForOrchestration adds an event to the active queue for
// asynchronous rule matching. It is a no-op when the worker has not been
// started or has been stopped.
func EnqueueEventForOrchestration(event eventModel.Event) {
	queueMu.RLock()
	q := activeQueue
	queueMu.RUnlock()
	if q == nil {
		return
	}
	if err := q.Enqueue(event); err != nil {
		log.GetLogger().Error(fmt.Sprintf(
			"worker: failed to enqueue event %s for orchestration: %v", event.EventId, err))
	}
}

// StopOrchestrationWorker gracefully shuts down the orchestration queue. It
// nils out the global reference under a write lock before calling Close,
// ensuring no concurrent Enqueue can send on a closed queue. It should be
// called during application shutdown.
func StopOrchestrationWorker() error {
	queueMu.Lock()
	q := activeQueue
	activeQueue = nil
	queueMu.Unlock()
	if q != nil {
		return q.Close()
	}
	return nil
}

// processEvent matches the event against the org's active rules and, for
// every rule whose conditions pass, runs its actions in order. Multiple
// rules can match the same event (unlike unification rules); they run in
// ascending priority order because the store query already sorts them.
func processEvent(event eventModel.Event) {

	logger := log.GetLogger()

	rules, err := orchestrationStore.GetActiveRulesForTrigger(event.OrgHandle, event.EventType, event.EventName)
	if err != nil {
		logger.Error(fmt.Sprintf("worker: failed to fetch orchestration rules for event %s", event.EventId), log.Error(err))
		return
	}
	if len(rules) == 0 {
		return
	}

	profile := fetchProfileAsMap(event.ProfileId)

	for _, rule := range rules {
		execCtx := executor.NewExecutionContext(event.OrgHandle, rule.RuleId, event, profile)
		if !matcher.EvaluateConditions(rule.Conditions, execCtx.Data) {
			continue
		}
		for index, action := range rule.Actions {
			runAction(execCtx, rule, index, action)
		}
	}
}

// runAction executes a single action and records the outcome. One action
// failing does not stop the remaining actions in the rule or other rules
// from running — each is recorded independently.
func runAction(execCtx *executor.ExecutionContext, rule orchestrationModel.OrchestrationRule, index int, action orchestrationModel.Action) {

	logger := log.GetLogger()
	status := constants.ActionExecutionStatusSuccess
	var errMsg string

	if err := executor.Execute(execCtx, action); err != nil {
		status = constants.ActionExecutionStatusFailed
		errMsg = err.Error()
		logger.Error(fmt.Sprintf("worker: action[%d] (%s) failed for rule %s / event %s",
			index, action.Type, rule.RuleId, execCtx.Event.EventId), log.Error(err))
	}

	execution := orchestrationModel.ActionExecution{
		ExecutionId:  uuid.New().String(),
		RuleId:       rule.RuleId,
		EventId:      execCtx.Event.EventId,
		OrgHandle:    rule.OrgHandle,
		ActionIndex:  index,
		ActionType:   action.Type,
		Status:       status,
		AttemptCount: 1,
		ErrorMessage: errMsg,
		ExecutedAt:   time.Now().UTC(),
	}
	if err := orchestrationStore.AddActionExecution(execution); err != nil {
		logger.Error("worker: failed to record action execution", log.Error(err))
	}
}

// fetchProfileAsMap loads the profile the event belongs to (if any) as a
// plain map for condition evaluation and templating ("profile.traits.*",
// "profile.identity_attributes.*", ...). A missing or unresolvable profile
// is not an error — rules with no profile-scoped conditions still run.
func fetchProfileAsMap(profileId string) map[string]interface{} {
	if profileId == "" {
		return nil
	}
	profilesService := profileProvider.NewProfilesProvider().GetProfilesService()
	profileResp, err := profilesService.GetProfile(profileId)
	if err != nil || profileResp == nil {
		return nil
	}
	raw, err := json.Marshal(profileResp)
	if err != nil {
		return nil
	}
	var asMap map[string]interface{}
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return nil
	}
	return asMap
}
