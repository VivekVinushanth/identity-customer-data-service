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

// NotificationTemplate is reusable notify.email / notify.sms content,
// referenced from an orchestration action's config by TemplateId instead of
// inlining subject/body in every rule. Subject is only meaningful for the
// "email" channel; Body carries the email body or the SMS message text.
// Both Subject and Body may contain "{{event.*}}" / "{{profile.*}}" /
// "{{search_result.*}}" placeholders, resolved the same way action config is.
type NotificationTemplate struct {
	TemplateId string    `json:"template_id" bson:"template_id"`
	OrgHandle  string    `json:"org_handle" bson:"org_handle"`
	Channel    string    `json:"channel" bson:"channel" binding:"required"`
	Name       string    `json:"name" bson:"name" binding:"required"`
	Subject    string    `json:"subject,omitempty" bson:"subject,omitempty"`
	Body       string    `json:"body" bson:"body" binding:"required"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at"`
}

// NotificationTemplateRequest is the client-facing payload for creating a template.
type NotificationTemplateRequest struct {
	Channel string `json:"channel"`
	Name    string `json:"name"`
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body"`
}

// NotificationTemplateUpdateRequest is the client-facing payload for a partial
// (PATCH) update. Only non-nil fields are applied; Channel is immutable.
type NotificationTemplateUpdateRequest struct {
	Name    *string `json:"name,omitempty"`
	Subject *string `json:"subject,omitempty"`
	Body    *string `json:"body,omitempty"`
}
