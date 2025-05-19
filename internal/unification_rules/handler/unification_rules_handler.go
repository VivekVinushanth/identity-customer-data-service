/*
 * Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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
	"github.com/wso2/identity-customer-data-service/internal/system/utils"
	"github.com/wso2/identity-customer-data-service/internal/unification_rules/model"
	"github.com/wso2/identity-customer-data-service/internal/unification_rules/provider"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type UnificationRulesHandler struct {
	store map[string]model.UnificationRule
	mu    *sync.RWMutex
}

func NewUnificationRulesHandler() *UnificationRulesHandler {

	return &UnificationRulesHandler{
		store: make(map[string]model.UnificationRule),
		mu:    &sync.RWMutex{},
	}
}

// AddUnificationRule handles adding a new rule
func (urh *UnificationRulesHandler) AddUnificationRule(w http.ResponseWriter, r *http.Request) {
	var rule model.UnificationRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if rule.RuleId == "" {
		rule.RuleId = uuid.NewString()
	}
	ruleProvider := provider.NewUnificationRuleProvider()
	ruleService := ruleProvider.GetUnificationRuleService()
	err := ruleService.AddUnificationRule(rule)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rule)
}

// GetUnificationRules handles fetching all rules
func (urh *UnificationRulesHandler) GetUnificationRules(w http.ResponseWriter, r *http.Request) {
	ruleProvider := provider.NewUnificationRuleProvider()
	ruleService := ruleProvider.GetUnificationRuleService()
	rules, err := ruleService.GetUnificationRules()
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rules)
}

// GetUnificationRule Fetches a specific resolution rule.
func (urh *UnificationRulesHandler) GetUnificationRule(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ruleId := pathParts[len(pathParts)-1]
	ruleProvider := provider.NewUnificationRuleProvider()
	ruleService := ruleProvider.GetUnificationRuleService()
	rule, err := ruleService.GetUnificationRule(ruleId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rule)
}

// PatchUnificationRule applies partial updates to a unification rule.
func (urh *UnificationRulesHandler) PatchUnificationRule(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ruleId := pathParts[len(pathParts)-1]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	ruleProvider := provider.NewUnificationRuleProvider()
	ruleService := ruleProvider.GetUnificationRuleService()
	err := ruleService.PatchResolutionRule(ruleId, updates)
	if err != nil {
		utils.HandleError(w, err)
		return
	}

	rule, err := ruleService.GetUnificationRule(ruleId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rule)
}

// DeleteUnificationRule removes a resolution rule.
func (urh *UnificationRulesHandler) DeleteUnificationRule(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ruleId := pathParts[len(pathParts)-1]
	ruleProvider := provider.NewUnificationRuleProvider()
	ruleService := ruleProvider.GetUnificationRuleService()
	err := ruleService.DeleteUnificationRule(ruleId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
