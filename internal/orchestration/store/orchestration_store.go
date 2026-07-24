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

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/system/database/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/database/scripts"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
)

// AddOrchestrationRule persists a new orchestration rule.
func AddOrchestrationRule(rule model.OrchestrationRule) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		return dbError(errors2.ADD_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to get database client for adding orchestration rule: %s", rule.RuleName), err)
	}
	defer dbClient.Close()

	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return dbError(errors2.ADD_ORCHESTRATION_RULE, "Failed to serialize rule conditions", err)
	}
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return dbError(errors2.ADD_ORCHESTRATION_RULE, "Failed to serialize rule actions", err)
	}

	query := scripts.InsertOrchestrationRule[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, rule.RuleId, rule.OrgHandle, rule.RuleName, rule.Trigger.EventType,
		rule.Trigger.EventName, string(conditionsJSON), string(actionsJSON), rule.Priority, rule.IsActive,
		rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return dbError(errors2.ADD_ORCHESTRATION_RULE, fmt.Sprintf(
			"Error occurred while adding orchestration rule: %s", rule.RuleName), err)
	}

	logger.Info(fmt.Sprintf("Orchestration rule '%s' added successfully", rule.RuleName))
	return nil
}

// GetOrchestrationRules fetches all rules for an org, sorted by priority ascending.
func GetOrchestrationRules(orgHandle string) ([]model.OrchestrationRule, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return nil, dbError(errors2.GET_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to get database client for fetching orchestration rules for organization: %s", orgHandle), err)
	}
	defer dbClient.Close()

	query := scripts.GetOrchestrationRules[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, orgHandle)
	if err != nil {
		return nil, dbError(errors2.GET_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed in fetching orchestration rules for organization: %s", orgHandle), err)
	}
	return rowsToRules(results)
}

// GetActiveRulesForTrigger fetches active rules for an org whose trigger
// matches the given event_type/event_name, sorted by priority ascending.
func GetActiveRulesForTrigger(orgHandle, eventType, eventName string) ([]model.OrchestrationRule, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return nil, dbError(errors2.GET_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to get database client for matching orchestration rules for organization: %s", orgHandle), err)
	}
	defer dbClient.Close()

	query := scripts.GetActiveOrchestrationRulesForTrigger[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, orgHandle, eventType, eventName)
	if err != nil {
		return nil, dbError(errors2.GET_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed in matching orchestration rules for organization: %s", orgHandle), err)
	}
	return rowsToRules(results)
}

// GetOrchestrationRule fetches a specific rule by its Id.
func GetOrchestrationRule(ruleId string) (*model.OrchestrationRule, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return nil, dbError(errors2.GET_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to get database client for fetching orchestration rule: %s", ruleId), err)
	}
	defer dbClient.Close()

	query := scripts.GetOrchestrationRule[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, ruleId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, dbError(errors2.GET_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed in fetching orchestration rule with rule_id: %s", ruleId), err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	rules, err := rowsToRules(results)
	if err != nil {
		return nil, err
	}
	return &rules[0], nil
}

// UpdateOrchestrationRule replaces the mutable fields of an existing rule.
func UpdateOrchestrationRule(ruleId string, rule model.OrchestrationRule) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return dbError(errors2.UPDATE_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to get database client for updating orchestration rule: %s", ruleId), err)
	}
	defer dbClient.Close()

	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return dbError(errors2.UPDATE_ORCHESTRATION_RULE, "Failed to serialize rule conditions", err)
	}
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return dbError(errors2.UPDATE_ORCHESTRATION_RULE, "Failed to serialize rule actions", err)
	}

	query := scripts.UpdateOrchestrationRule[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, rule.RuleName, rule.Trigger.EventType, rule.Trigger.EventName,
		string(conditionsJSON), string(actionsJSON), rule.Priority, rule.IsActive, time.Now().UTC(), ruleId)
	if err != nil {
		return dbError(errors2.UPDATE_ORCHESTRATION_RULE, fmt.Sprintf(
			"Error occurred while updating orchestration rule for rule_id: %s", ruleId), err)
	}
	return nil
}

// DeleteOrchestrationRule deletes a rule by its Id.
func DeleteOrchestrationRule(ruleId string) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return dbError(errors2.DELETE_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to get database client for deleting orchestration rule: %s", ruleId), err)
	}
	defer dbClient.Close()

	query := scripts.DeleteOrchestrationRule[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, ruleId)
	if err != nil {
		return dbError(errors2.DELETE_ORCHESTRATION_RULE, fmt.Sprintf(
			"Failed to delete orchestration rule: %s", ruleId), err)
	}
	return nil
}

// AddActionExecution records the outcome of running one action.
func AddActionExecution(execution model.ActionExecution) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		logger.Error("Failed to get database client for recording action execution", log.Error(err))
		return err
	}
	defer dbClient.Close()

	query := scripts.InsertActionExecution[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, execution.ExecutionId, execution.RuleId, execution.EventId,
		execution.OrgHandle, execution.ActionIndex, execution.ActionType, execution.Status,
		execution.AttemptCount, nullableString(execution.ErrorMessage), execution.ExecutedAt)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to record action execution for rule %s", execution.RuleId), log.Error(err))
		return err
	}
	return nil
}

// GetActionExecutionsForRule fetches the most recent action executions for a rule, newest first.
func GetActionExecutionsForRule(ruleId string, limit int) ([]model.ActionExecution, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	if err != nil {
		return nil, dbError(errors2.GET_ACTION_EXECUTIONS, fmt.Sprintf(
			"Failed to get database client for fetching action executions for rule: %s", ruleId), err)
	}
	defer dbClient.Close()

	query := scripts.GetActionExecutionsForRule[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, ruleId, limit)
	if err != nil {
		return nil, dbError(errors2.GET_ACTION_EXECUTIONS, fmt.Sprintf(
			"Failed in fetching action executions for rule: %s", ruleId), err)
	}

	executions := make([]model.ActionExecution, 0, len(results))
	for _, row := range results {
		var execution model.ActionExecution
		execution.ExecutionId, _ = row["execution_id"].(string)
		execution.RuleId, _ = row["rule_id"].(string)
		execution.EventId, _ = row["event_id"].(string)
		execution.OrgHandle, _ = row["org_handle"].(string)
		if idx, ok := row["action_index"].(int64); ok {
			execution.ActionIndex = int(idx)
		}
		execution.ActionType, _ = row["action_type"].(string)
		execution.Status, _ = row["status"].(string)
		if attempts, ok := row["attempt_count"].(int64); ok {
			execution.AttemptCount = int(attempts)
		}
		execution.ErrorMessage, _ = row["error_message"].(string)
		if executedAt, ok := row["executed_at"].(time.Time); ok {
			execution.ExecutedAt = executedAt
		}
		executions = append(executions, execution)
	}
	return executions, nil
}

func rowsToRules(results []map[string]interface{}) ([]model.OrchestrationRule, error) {

	rules := make([]model.OrchestrationRule, 0, len(results))
	for _, row := range results {
		var rule model.OrchestrationRule
		rule.RuleId, _ = row["rule_id"].(string)
		rule.OrgHandle, _ = row["org_handle"].(string)
		rule.RuleName, _ = row["rule_name"].(string)
		rule.Trigger.EventType, _ = row["event_type"].(string)
		rule.Trigger.EventName, _ = row["event_name"].(string)
		if priority, ok := row["priority"].(int64); ok {
			rule.Priority = int(priority)
		}
		if isActive, ok := row["is_active"].(bool); ok {
			rule.IsActive = isActive
		}
		if createdAt, ok := row["created_at"].(time.Time); ok {
			rule.CreatedAt = createdAt
		}
		if updatedAt, ok := row["updated_at"].(time.Time); ok {
			rule.UpdatedAt = updatedAt
		}
		if raw, ok := row["conditions"].(string); ok && raw != "" {
			if err := json.Unmarshal([]byte(raw), &rule.Conditions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal conditions for rule %s: %w", rule.RuleId, err)
			}
		}
		if raw, ok := row["actions"].(string); ok && raw != "" {
			if err := json.Unmarshal([]byte(raw), &rule.Actions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal actions for rule %s: %w", rule.RuleId, err)
			}
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func dbError(base errors2.ErrorMessage, description string, cause error) error {
	log.GetLogger().Debug(description, log.Error(cause))
	return errors2.NewServerError(errors2.ErrorMessage{
		Code:        base.Code,
		Message:     base.Message,
		Description: description,
	}, cause)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
