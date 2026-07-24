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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/matcher"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/system/config"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

func init() {
	Register(constants.ActionTypeNotifySMS, &notifySMSExecutor{
		client: &http.Client{Timeout: 10 * time.Second},
	})
}

// notifySMSExecutor is a generic HTTP relay: there is no single standard SMS
// gateway API, so this POSTs {"to", "message"} as JSON to a configured
// provider URL — point it at your gateway directly (if it accepts a simple
// JSON webhook) or at a thin adapter in front of it (e.g. Twilio, Vonage).
// Config:
//
//	{ "to": "{{profile.identity_attributes.mobile}}" (required),
//	  "template_id": "..." (optional, references a NotificationTemplate of
//	    channel "sms"), "message": "...",
//	  "provider_url": "..." (optional; overrides notifications.sms.provider_url) }
//
// Either template_id or message is required; message (if set) overrides the
// referenced template's body.
type notifySMSExecutor struct {
	client *http.Client
}

func (e *notifySMSExecutor) Execute(execCtx *ExecutionContext, action model.Action) error {

	smsCfg := config.GetCDSRuntime().Config.Notifications.SMS

	providerURL, _ := action.Config["provider_url"].(string)
	if providerURL == "" {
		providerURL = smsCfg.ProviderURL
	}
	if providerURL == "" {
		return fmt.Errorf("notify.sms: no SMS provider configured " +
			"(set notifications.sms.provider_url or config.provider_url on the action)")
	}

	to, _ := action.Config["to"].(string)
	to = matcher.ResolveTemplate(to, execCtx.Data)
	if to == "" {
		return fmt.Errorf("notify.sms: config.to is required and must resolve to a non-empty destination")
	}

	_, message, err := resolveNotificationContent(execCtx, action, constants.NotificationChannelSMS)
	if err != nil {
		return fmt.Errorf("notify.sms: %w", err)
	}

	body, err := json.Marshal(map[string]string{"to": to, "message": message})
	if err != nil {
		return fmt.Errorf("notify.sms: failed to serialize payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, providerURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notify.sms: invalid provider_url %q: %w", providerURL, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if smsCfg.AuthHeader != "" {
		req.Header.Set("Authorization", smsCfg.AuthHeader)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify.sms: request to %s failed: %w", providerURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notify.sms: provider %s returned status %d", providerURL, resp.StatusCode)
	}
	return nil
}
