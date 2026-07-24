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

// Package executor implements the pluggable action executors that back an
// orchestration rule's Actions[]. Built-in action types register themselves
// via init() in their own file (see profile_actions.go, callback.go,
// notify_email.go, notify_sms.go), following the same registry pattern as
// internal/system/queue/factory.go. A deployment can add its own action type
// by calling Register from an init() in a package imported (blank or
// otherwise) by cmd/server/main.go.
package executor

import (
	"fmt"
	"sync"

	eventModel "github.com/wso2/identity-customer-data-service/internal/event/model"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
)

// ExecutionContext carries everything an executor needs: the org the rule
// belongs to, the triggering event, and the shared, mutable data map used for
// condition evaluation and template resolution ("event", "profile" and,
// once populated by an earlier profile.search action, "search_result").
type ExecutionContext struct {
	OrgHandle string
	RuleId    string
	Event     eventModel.Event
	Data      map[string]interface{}
}

// NewExecutionContext builds an ExecutionContext with the event and (if
// found) profile already populated into Data.
func NewExecutionContext(orgHandle, ruleId string, event eventModel.Event, profile map[string]interface{}) *ExecutionContext {
	data := map[string]interface{}{
		"event": event.ToMap(),
	}
	if profile != nil {
		data["profile"] = profile
	}
	return &ExecutionContext{
		OrgHandle: orgHandle,
		RuleId:    ruleId,
		Event:     event,
		Data:      data,
	}
}

// SetContextValue stores a named result (e.g. "search_result") so later
// actions in the same rule can reference it via {{search_result.*}}.
func (c *ExecutionContext) SetContextValue(key string, value map[string]interface{}) {
	c.Data[key] = value
}

// Executor runs a single action. It returns an error when the action failed;
// the caller (the orchestration worker) is responsible for recording the
// outcome as an ActionExecution audit row.
type Executor interface {
	Execute(execCtx *ExecutionContext, action model.Action) error
}

var (
	mu        sync.RWMutex
	executors = map[string]Executor{}
)

// Register makes an Executor available under the given action type. Call
// this from an init() function so the executor is available as soon as its
// package is imported.
func Register(actionType string, e Executor) {
	mu.Lock()
	defer mu.Unlock()
	executors[actionType] = e
}

// Get returns the Executor registered for actionType, if any.
func Get(actionType string) (Executor, bool) {
	mu.RLock()
	defer mu.RUnlock()
	e, ok := executors[actionType]
	return e, ok
}

// Execute looks up the executor for action.Type and runs it. An unknown
// action type is a configuration error, not a transient failure.
func Execute(execCtx *ExecutionContext, action model.Action) error {
	e, ok := Get(action.Type)
	if !ok {
		return fmt.Errorf("executor: no executor registered for action type %q", action.Type)
	}
	return e.Execute(execCtx, action)
}
