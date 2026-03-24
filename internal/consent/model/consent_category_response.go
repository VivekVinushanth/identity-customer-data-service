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

package models

// ConsentCategoryResponse is the API-facing representation of a consent category.
//
// Non-applicationData attributes are listed as plain strings under "attributes"
// (e.g. "traits.engagement_score", "identity_attributes.email"). The scope is
// always derivable from the attribute name prefix so it is not repeated.
//
// applicationData attributes are grouped by app_id under "application_data"
// (e.g. {"app123": ["application_data.certifications", "application_data.learning_paths"]}).
//
// org_handle is intentionally omitted — it is an internal field not relevant to callers.
type ConsentCategoryResponse struct {
	CategoryName       string                 `json:"category_name"`
	CategoryIdentifier string                 `json:"category_identifier"`
	Purpose            string                 `json:"purpose"`
	Destinations       []string               `json:"destinations,omitempty"`
	Attributes         []string               `json:"attributes,omitempty"`
	ApplicationData    map[string][]string    `json:"application_data,omitempty"`
	IsMandatory        bool                   `json:"is_mandatory"`
}

// ToResponse converts an internal ConsentCategory to its API response form.
func (c ConsentCategory) ToResponse() ConsentCategoryResponse {
	attrs := make([]string, 0)
	appData := make(map[string][]string)

	for _, a := range c.Attributes {
		if a.AppId != "" {
			appData[a.AppId] = append(appData[a.AppId], a.AttributeName)
		} else {
			attrs = append(attrs, a.AttributeName)
		}
	}

	resp := ConsentCategoryResponse{
		CategoryName:       c.CategoryName,
		CategoryIdentifier: c.CategoryIdentifier,
		Purpose:            c.Purpose,
		Destinations:       c.Destinations,
		IsMandatory:        c.IsMandatory,
	}
	if len(attrs) > 0 {
		resp.Attributes = attrs
	}
	if len(appData) > 0 {
		resp.ApplicationData = appData
	}
	return resp
}
