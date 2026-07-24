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
	"net/smtp"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/matcher"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/system/config"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

func init() {
	Register(constants.ActionTypeNotifyEmail, &notifyEmailExecutor{})
}

// notifyEmailExecutor sends a plain-text email via the SMTP server
// configured under notifications.smtp in deployment.yaml. Config:
//
//	{ "to": "{{profile.identity_attributes.emailaddress}}" (required),
//	  "template_id": "..." (optional, references a NotificationTemplate of
//	    channel "email"), "subject": "...", "body": "..." }
//
// Either template_id or body is required; subject/body (if set) override the
// referenced template's values. All may contain
// "{{event.*}}"/"{{profile.*}}"/"{{search_result.*}}" placeholders.
type notifyEmailExecutor struct{}

func (e *notifyEmailExecutor) Execute(execCtx *ExecutionContext, action model.Action) error {

	smtpCfg := config.GetCDSRuntime().Config.Notifications.SMTP
	if smtpCfg.Host == "" {
		return fmt.Errorf("notify.email: no SMTP server configured (notifications.smtp.host is empty)")
	}

	to, _ := action.Config["to"].(string)
	to = matcher.ResolveTemplate(to, execCtx.Data)
	if to == "" {
		return fmt.Errorf("notify.email: config.to is required and must resolve to a non-empty address")
	}

	subject, body, err := resolveNotificationContent(execCtx, action, constants.NotificationChannelEmail)
	if err != nil {
		return fmt.Errorf("notify.email: %w", err)
	}

	from := smtpCfg.FromAddress
	if from == "" {
		from = smtpCfg.Username
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)
	var auth smtp.Auth
	if smtpCfg.Username != "" {
		auth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)
	}

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("notify.email: failed to send to %s: %w", to, err)
	}
	return nil
}
