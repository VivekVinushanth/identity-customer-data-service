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

	consentModel "github.com/wso2/identity-customer-data-service/internal/consent/model"
	consentStore "github.com/wso2/identity-customer-data-service/internal/consent/store"
	"github.com/wso2/identity-customer-data-service/internal/profile/model"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

// FilterProfileByConsent returns a ProfileResponse filtered to only the attributes
// the profile owner has consented to across all requested consent categories.
//
// If consentIds is empty, only core fields (profile_id, user_id, meta) are returned.
// For multiple consentIds the intersection of allowed attribute sets is applied —
// an attribute is included only if it appears in every consented category's attribute list.
func FilterProfileByConsent(response model.ProfileResponse, profileId string, consentIds []string) (model.ProfileResponse, error) {

	filtered := model.ProfileResponse{
		ProfileId:  response.ProfileId,
		UserId:     response.UserId,
		Meta:       response.Meta,
		MergedTo:   response.MergedTo,
		MergedFrom: response.MergedFrom,
	}

	if len(consentIds) == 0 {
		return filtered, nil
	}

	// Fetch attributes for categories the profile has actively consented to.
	attrsByCategory, err := consentStore.GetConsentedCategoryAttributesByProfileId(profileId, consentIds)
	if err != nil {
		return filtered, err
	}

	// Build per-category attribute key sets, then compute the intersection.
	// An attribute key has the form "<scope>::<attributeId>" or
	// "applicationData::<appId>::<attributeId>" for app-scoped attributes.
	categorySets := make([]map[string]bool, 0, len(consentIds))
	for _, id := range consentIds {
		attrs, ok := attrsByCategory[id]
		if !ok {
			// Profile has not consented to this category — contributes empty set,
			// which makes the intersection empty for any attribute gated by it.
			categorySets = append(categorySets, map[string]bool{})
			continue
		}
		set := make(map[string]bool, len(attrs))
		for _, attr := range attrs {
			set[attributeKey(attr)] = true
		}
		categorySets = append(categorySets, set)
	}

	allowed := unionSets(categorySets)
	if len(allowed) == 0 {
		return filtered, nil
	}

	// Filter identityAttributes
	if len(response.IdentityAttributes) > 0 {
		ia := make(map[string]interface{})
		for k, v := range response.IdentityAttributes {
			if allowed[fmt.Sprintf("%s::%s", constants.ScopeIdentityAttributes, k)] {
				ia[k] = v
			}
		}
		if len(ia) > 0 {
			filtered.IdentityAttributes = ia
		}
	}

	// Filter traits
	if len(response.Traits) > 0 {
		tr := make(map[string]interface{})
		for k, v := range response.Traits {
			if allowed[fmt.Sprintf("%s::%s", constants.ScopeTraits, k)] {
				tr[k] = v
			}
		}
		if len(tr) > 0 {
			filtered.Traits = tr
		}
	}

	// Filter applicationData — per-app, per-attribute
	if len(response.ApplicationData) > 0 {
		appData := make(map[string]map[string]interface{})
		for appId, attrs := range response.ApplicationData {
			filteredAttrs := make(map[string]interface{})
			for k, v := range attrs {
				if allowed[fmt.Sprintf("%s::%s::%s", constants.ScopeApplicationData, appId, k)] {
					filteredAttrs[k] = v
				}
			}
			if len(filteredAttrs) > 0 {
				appData[appId] = filteredAttrs
			}
		}
		if len(appData) > 0 {
			filtered.ApplicationData = appData
		}
	}

	return filtered, nil
}

// attributeKey returns the canonical string key for a ConsentAttribute.
func attributeKey(attr consentModel.ConsentAttribute) string {
	if attr.Scope == constants.ScopeApplicationData {
		return fmt.Sprintf("%s::%s::%s", attr.Scope, attr.AppId, attr.AttributeId)
	}
	return fmt.Sprintf("%s::%s", attr.Scope, attr.AttributeId)
}

// unionSets returns the union of all provided sets.
func unionSets(sets []map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for _, s := range sets {
		for k := range s {
			result[k] = true
		}
	}
	return result
}
