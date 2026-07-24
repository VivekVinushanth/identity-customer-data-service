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

package service

import (
	"fmt"
	"net/http"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/store"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
)

type OrchestrationRuleServiceInterface interface {
	AddOrchestrationRule(rule model.OrchestrationRule) error
	GetOrchestrationRules(orgHandle string) ([]model.OrchestrationRule, error)
	GetOrchestrationRule(ruleId string) (*model.OrchestrationRule, error)
	UpdateOrchestrationRule(ruleId string, rule model.OrchestrationRule) error
	DeleteOrchestrationRule(ruleId string) error
	GetActionExecutions(ruleId string, limit int) ([]model.ActionExecution, error)
}

// OrchestrationRuleService is the default implementation of OrchestrationRuleServiceInterface.
type OrchestrationRuleService struct{}

// GetOrchestrationRuleService creates a new instance of OrchestrationRuleService.
func GetOrchestrationRuleService() OrchestrationRuleServiceInterface {

	return &OrchestrationRuleService{}
}

// AddOrchestrationRule validates and persists a new orchestration rule.
func (s *OrchestrationRuleService) AddOrchestrationRule(rule model.OrchestrationRule) error {

	if err := validateRule(rule); err != nil {
		return err
	}

	existingRules, err := store.GetOrchestrationRules(rule.OrgHandle)
	if err != nil {
		return err
	}
	for _, existing := range existingRules {
		if existing.Priority == rule.Priority {
			return errors2.NewClientError(errors2.ErrorMessage{
				Code:        errors2.ORCHESTRATION_RULE_PRIORITY_EXISTS.Code,
				Message:     errors2.ORCHESTRATION_RULE_PRIORITY_EXISTS.Message,
				Description: "Orchestration rule with same priority exists.",
			}, http.StatusBadRequest)
		}
	}

	return store.AddOrchestrationRule(rule)
}

// GetOrchestrationRules fetches all orchestration rules for an org.
func (s *OrchestrationRuleService) GetOrchestrationRules(orgHandle string) ([]model.OrchestrationRule, error) {
	return store.GetOrchestrationRules(orgHandle)
}

// GetOrchestrationRule fetches a specific rule.
func (s *OrchestrationRuleService) GetOrchestrationRule(ruleId string) (*model.OrchestrationRule, error) {

	rule, err := store.GetOrchestrationRule(ruleId)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.ORCHESTRATION_RULE_NOT_FOUND.Code,
			Message:     errors2.ORCHESTRATION_RULE_NOT_FOUND.Message,
			Description: fmt.Sprintf("Orchestration rule: '%s' not found", ruleId),
		}, http.StatusNotFound)
	}
	return rule, nil
}

// UpdateOrchestrationRule replaces the mutable fields of an existing rule.
func (s *OrchestrationRuleService) UpdateOrchestrationRule(ruleId string, rule model.OrchestrationRule) error {

	if err := validateRule(rule); err != nil {
		return err
	}

	existingRules, err := store.GetOrchestrationRules(rule.OrgHandle)
	if err != nil {
		return err
	}
	for _, existing := range existingRules {
		if existing.RuleId != ruleId && existing.Priority == rule.Priority {
			return errors2.NewClientError(errors2.ErrorMessage{
				Code:        errors2.ORCHESTRATION_RULE_PRIORITY_EXISTS.Code,
				Message:     errors2.ORCHESTRATION_RULE_PRIORITY_EXISTS.Message,
				Description: "Orchestration rule with same priority exists.",
			}, http.StatusBadRequest)
		}
	}

	return store.UpdateOrchestrationRule(ruleId, rule)
}

// DeleteOrchestrationRule removes a rule.
func (s *OrchestrationRuleService) DeleteOrchestrationRule(ruleId string) error {
	return store.DeleteOrchestrationRule(ruleId)
}

// GetActionExecutions fetches the audit trail for a rule's action runs.
func (s *OrchestrationRuleService) GetActionExecutions(ruleId string, limit int) ([]model.ActionExecution, error) {

	if limit <= 0 {
		limit = constants.DefaultLimit
	}
	return store.GetActionExecutionsForRule(ruleId, limit)
}

func validateRule(rule model.OrchestrationRule) error {

	if rule.RuleName == "" {
		return errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.ORCHESTRATION_RULE_VALIDATION.Code,
			Message:     errors2.ORCHESTRATION_RULE_VALIDATION.Message,
			Description: "rule_name is required",
		}, http.StatusBadRequest)
	}
	if rule.Trigger.EventType == "" || rule.Trigger.EventName == "" {
		return errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.ORCHESTRATION_RULE_VALIDATION.Code,
			Message:     errors2.ORCHESTRATION_RULE_VALIDATION.Message,
			Description: "trigger.event_type and trigger.event_name are required",
		}, http.StatusBadRequest)
	}
	if len(rule.Actions) == 0 {
		return errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.ORCHESTRATION_RULE_VALIDATION.Code,
			Message:     errors2.ORCHESTRATION_RULE_VALIDATION.Message,
			Description: "at least one action is required",
		}, http.StatusBadRequest)
	}
	for i, action := range rule.Actions {
		if !constants.AllowedActionTypes[action.Type] {
			return errors2.NewClientError(errors2.ErrorMessage{
				Code:        errors2.ORCHESTRATION_RULE_VALIDATION.Code,
				Message:     errors2.ORCHESTRATION_RULE_VALIDATION.Message,
				Description: fmt.Sprintf("actions[%d].type %q is not a supported action type", i, action.Type),
			}, http.StatusBadRequest)
		}
	}
	for i, cond := range rule.Conditions {
		if !constants.AllowedConditionOperators[cond.Operator] {
			return errors2.NewClientError(errors2.ErrorMessage{
				Code:        errors2.ORCHESTRATION_RULE_VALIDATION.Code,
				Message:     errors2.ORCHESTRATION_RULE_VALIDATION.Message,
				Description: fmt.Sprintf("conditions[%d].operator %q is not a supported operator", i, cond.Operator),
			}, http.StatusBadRequest)
		}
	}
	return nil
}
