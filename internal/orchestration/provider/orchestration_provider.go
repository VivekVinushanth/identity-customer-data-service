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

package provider

import (
	"github.com/wso2/identity-customer-data-service/internal/orchestration/service"
)

// OrchestrationRuleProviderInterface defines the interface for the orchestration rule provider.
type OrchestrationRuleProviderInterface interface {
	GetOrchestrationRuleService() service.OrchestrationRuleServiceInterface
}

// OrchestrationRuleProvider is the default implementation of OrchestrationRuleProviderInterface.
type OrchestrationRuleProvider struct{}

// NewOrchestrationRuleProvider creates a new instance of OrchestrationRuleProvider.
func NewOrchestrationRuleProvider() OrchestrationRuleProviderInterface {

	return &OrchestrationRuleProvider{}
}

// GetOrchestrationRuleService returns the orchestration rule service instance.
func (p *OrchestrationRuleProvider) GetOrchestrationRuleService() service.OrchestrationRuleServiceInterface {

	return service.GetOrchestrationRuleService()
}
