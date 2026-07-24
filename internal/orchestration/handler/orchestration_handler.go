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

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	adminConfigService "github.com/wso2/identity-customer-data-service/internal/admin_config/service"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/security"
	"github.com/wso2/identity-customer-data-service/internal/system/utils"
)

type OrchestrationRuleHandler struct{}

func NewOrchestrationRuleHandler() *OrchestrationRuleHandler {

	return &OrchestrationRuleHandler{}
}

// AddOrchestrationRule handles POST /orchestration-rules.
func (h *OrchestrationRuleHandler) AddOrchestrationRule(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "orchestration_rules:create"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	var req model.OrchestrationRuleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		clientError := errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.BAD_REQUEST.Code,
			Message:     errors2.BAD_REQUEST.Message,
			Description: utils.HandleDecodeError(err, "orchestration rule"),
		}, http.StatusBadRequest)
		utils.WriteErrorResponse(w, clientError)
		return
	}

	now := time.Now().UTC()
	rule := model.OrchestrationRule{
		RuleId:     uuid.New().String(),
		OrgHandle:  orgHandle,
		RuleName:   req.RuleName,
		Trigger:    req.Trigger,
		Conditions: req.Conditions,
		Actions:    req.Actions,
		Priority:   req.Priority,
		IsActive:   req.IsActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	ruleService := provider.NewOrchestrationRuleProvider().GetOrchestrationRuleService()
	if err := ruleService.AddOrchestrationRule(rule); err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusCreated, rule, constants.OrchestrationRuleResource)
}

// GetOrchestrationRules handles GET /orchestration-rules.
func (h *OrchestrationRuleHandler) GetOrchestrationRules(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "orchestration_rules:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	ruleService := provider.NewOrchestrationRuleProvider().GetOrchestrationRuleService()
	rules, err := ruleService.GetOrchestrationRules(orgHandle)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, rules, constants.OrchestrationRuleResource)
}

// GetOrchestrationRule handles GET /orchestration-rules/{ruleId}.
func (h *OrchestrationRuleHandler) GetOrchestrationRule(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "orchestration_rules:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	ruleId := r.PathValue("ruleId")
	if ruleId == "" {
		utils.HandleError(w, notFoundError(ruleId))
		return
	}

	ruleService := provider.NewOrchestrationRuleProvider().GetOrchestrationRuleService()
	rule, err := ruleService.GetOrchestrationRule(ruleId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, rule, constants.OrchestrationRuleResource)
}

// PatchOrchestrationRule handles PATCH /orchestration-rules/{ruleId}.
func (h *OrchestrationRuleHandler) PatchOrchestrationRule(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "orchestration_rules:update"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	ruleId := r.PathValue("ruleId")
	if ruleId == "" {
		utils.HandleError(w, notFoundError(ruleId))
		return
	}

	var patch model.OrchestrationRuleUpdateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&patch); err != nil {
		clientError := errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.BAD_REQUEST.Code,
			Message:     errors2.BAD_REQUEST.Message,
			Description: utils.HandleDecodeError(err, "orchestration rule"),
		}, http.StatusBadRequest)
		utils.WriteErrorResponse(w, clientError)
		return
	}

	ruleService := provider.NewOrchestrationRuleProvider().GetOrchestrationRuleService()
	rule, err := ruleService.GetOrchestrationRule(ruleId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}

	if patch.RuleName != nil {
		rule.RuleName = *patch.RuleName
	}
	if patch.Trigger != nil {
		rule.Trigger = *patch.Trigger
	}
	if patch.Conditions != nil {
		rule.Conditions = *patch.Conditions
	}
	if patch.Actions != nil {
		rule.Actions = *patch.Actions
	}
	if patch.Priority != nil {
		rule.Priority = *patch.Priority
	}
	if patch.IsActive != nil {
		rule.IsActive = *patch.IsActive
	}

	if err := ruleService.UpdateOrchestrationRule(ruleId, *rule); err != nil {
		utils.HandleError(w, err)
		return
	}

	updated, err := ruleService.GetOrchestrationRule(ruleId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, updated, constants.OrchestrationRuleResource)
}

// DeleteOrchestrationRule handles DELETE /orchestration-rules/{ruleId}.
func (h *OrchestrationRuleHandler) DeleteOrchestrationRule(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "orchestration_rules:delete"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	ruleId := r.PathValue("ruleId")
	if ruleId == "" {
		utils.HandleError(w, notFoundError(ruleId))
		return
	}

	ruleService := provider.NewOrchestrationRuleProvider().GetOrchestrationRuleService()
	if err := ruleService.DeleteOrchestrationRule(ruleId); err != nil {
		utils.HandleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetActionExecutions handles GET /orchestration-rules/{ruleId}/executions —
// the audit trail used to debug "why didn't my action fire".
func (h *OrchestrationRuleHandler) GetActionExecutions(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "orchestration_rules:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	ruleId := r.PathValue("ruleId")
	if ruleId == "" {
		utils.HandleError(w, notFoundError(ruleId))
		return
	}

	limit := constants.DefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ruleService := provider.NewOrchestrationRuleProvider().GetOrchestrationRuleService()
	executions, err := ruleService.GetActionExecutions(ruleId, limit)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, executions, constants.ActionExecutionResource)
}

func isCDSEnabled(orgHandle string) bool {
	return adminConfigService.GetAdminConfigService().IsCDSEnabled(orgHandle)
}

func cdsNotEnabledError() *errors2.ClientError {
	return errors2.NewClientError(errors2.ErrorMessage{
		Code:        errors2.CDS_NOT_ENABLED.Code,
		Message:     errors2.CDS_NOT_ENABLED.Message,
		Description: errors2.CDS_NOT_ENABLED.Description,
	}, http.StatusBadRequest)
}

func notFoundError(ruleId string) *errors2.ClientError {
	return errors2.NewClientError(errors2.ErrorMessage{
		Code:        errors2.ORCHESTRATION_RULE_NOT_FOUND.Code,
		Message:     errors2.ORCHESTRATION_RULE_NOT_FOUND.Message,
		Description: "Invalid path for orchestration rule retrieval",
	}, http.StatusNotFound)
}
