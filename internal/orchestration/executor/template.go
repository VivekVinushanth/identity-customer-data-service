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

package executor

import (
	"fmt"

	notificationTemplateProvider "github.com/wso2/identity-customer-data-service/internal/notification_template/provider"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/matcher"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
)

// resolveNotificationContent resolves the subject/body a notify.email or
// notify.sms action should send, following this precedence:
//
//  1. config.template_id, if set, is loaded (and must be a template of the
//     given channel) and supplies the base subject/body.
//  2. config.subject / config.body, if set, override the template's values
//     (or stand alone when no template_id is given).
//
// At least one of template_id or body must be present. The result is
// template-resolved ("{{event.*}}"/"{{profile.*}}"/"{{search_result.*}}")
// against execCtx before being returned.
func resolveNotificationContent(execCtx *ExecutionContext, action model.Action, channel string) (subject string, body string, err error) {

	if templateId, ok := action.Config["template_id"].(string); ok && templateId != "" {
		templateService := notificationTemplateProvider.NewNotificationTemplateProvider().GetNotificationTemplateService()
		template, tErr := templateService.GetNotificationTemplate(templateId)
		if tErr != nil {
			return "", "", fmt.Errorf("failed to load template %q: %w", templateId, tErr)
		}
		if template.Channel != channel {
			return "", "", fmt.Errorf("template %q is a %q template, not %q", templateId, template.Channel, channel)
		}
		subject, body = template.Subject, template.Body
	}

	if configSubject, ok := action.Config["subject"].(string); ok && configSubject != "" {
		subject = configSubject
	}
	if configBody, ok := action.Config["body"].(string); ok && configBody != "" {
		body = configBody
	}
	if configMessage, ok := action.Config["message"].(string); ok && configMessage != "" {
		body = configMessage
	}

	if body == "" {
		return "", "", fmt.Errorf("config.template_id (referencing an existing %s template) or config.body/message is required", channel)
	}

	return matcher.ResolveTemplate(subject, execCtx.Data), matcher.ResolveTemplate(body, execCtx.Data), nil
}
