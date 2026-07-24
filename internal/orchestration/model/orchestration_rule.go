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

package model

import "time"

// Trigger selects which events a rule reacts to. A rule only becomes a
// candidate for evaluation when an incoming event's EventType and EventName
// both match.
type Trigger struct {
	EventType string `json:"event_type" binding:"required"`
	EventName string `json:"event_name" binding:"required"`
}

// Condition is a single predicate evaluated against the triggering event and
// the profile it belongs to. Field is a dotted path prefixed with "event." or
// "profile." (e.g. "event.properties.value", "profile.traits.plan"); Operator
// is one of the constants.AllowedConditionOperators.
type Condition struct {
	Field    string `json:"field" binding:"required"`
	Operator string `json:"operator" binding:"required"`
	Value    string `json:"value"`
}

// Action is one step of a rule's execution. Type selects the executor (see
// constants.AllowedActionTypes); Config is executor-specific and may contain
// "{{event.*}}" / "{{profile.*}}" / "{{search_result.*}}" template
// placeholders that are resolved against the execution context at run time.
type Action struct {
	Type   string                 `json:"type" binding:"required"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// OrchestrationRule reacts to a triggering event: if Conditions all pass
// against the event and the profile it belongs to, Actions run in order.
// Multiple rules can match the same event; within an org they run in
// ascending Priority order.
type OrchestrationRule struct {
	RuleId     string      `json:"rule_id" bson:"rule_id"`
	OrgHandle  string      `json:"org_handle" bson:"org_handle"`
	RuleName   string      `json:"rule_name" bson:"rule_name" binding:"required"`
	Trigger    Trigger     `json:"trigger" bson:"trigger" binding:"required"`
	Conditions []Condition `json:"conditions,omitempty" bson:"conditions,omitempty"`
	Actions    []Action    `json:"actions" bson:"actions" binding:"required"`
	Priority   int         `json:"priority" bson:"priority"`
	IsActive   bool        `json:"is_active" bson:"is_active"`
	CreatedAt  time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" bson:"updated_at"`
}

// OrchestrationRuleRequest is the client-facing payload for creating a rule.
type OrchestrationRuleRequest struct {
	RuleName   string      `json:"rule_name"`
	Trigger    Trigger     `json:"trigger"`
	Conditions []Condition `json:"conditions,omitempty"`
	Actions    []Action    `json:"actions"`
	Priority   int         `json:"priority"`
	IsActive   bool        `json:"is_active"`
}

// OrchestrationRuleUpdateRequest is the client-facing payload for a partial
// (PATCH) update. Only non-nil fields are applied.
type OrchestrationRuleUpdateRequest struct {
	RuleName   *string      `json:"rule_name,omitempty"`
	Trigger    *Trigger     `json:"trigger,omitempty"`
	Conditions *[]Condition `json:"conditions,omitempty"`
	Actions    *[]Action    `json:"actions,omitempty"`
	Priority   *int         `json:"priority,omitempty"`
	IsActive   *bool        `json:"is_active,omitempty"`
}

// ActionExecution is one audit row recording the outcome of running a single
// action from a rule against a single event.
type ActionExecution struct {
	ExecutionId  string    `json:"execution_id" bson:"execution_id"`
	RuleId       string    `json:"rule_id" bson:"rule_id"`
	EventId      string    `json:"event_id" bson:"event_id"`
	OrgHandle    string    `json:"org_handle" bson:"org_handle"`
	ActionIndex  int       `json:"action_index" bson:"action_index"`
	ActionType   string    `json:"action_type" bson:"action_type"`
	Status       string    `json:"status" bson:"status"`
	AttemptCount int       `json:"attempt_count" bson:"attempt_count"`
	ErrorMessage string    `json:"error_message,omitempty" bson:"error_message,omitempty"`
	ExecutedAt   time.Time `json:"executed_at" bson:"executed_at"`
}
