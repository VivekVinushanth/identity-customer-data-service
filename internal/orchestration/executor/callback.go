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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/matcher"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

func init() {
	Register(constants.ActionTypeAppCallback, &appCallbackExecutor{
		client: &http.Client{Timeout: 10 * time.Second},
	})
}

// appCallbackExecutor POSTs a JSON payload to an application-owned endpoint —
// this is the "prompt something / show a banner" hook: the application
// receives the event + profile and decides what to render or do next.
//
// Config:
//
//	{ "endpoint_url": "https://app.example.com/cds/callback" (required, may
//	    contain "{{event.*}}"/"{{profile.*}}"/"{{search_result.*}}" placeholders),
//	  "secret": "..." (optional; if set, the request is signed — see below),
//	  "payload": { ... } (optional; defaults to {rule_id, event, profile}),
//	  "headers": { "X-Custom": "..." } (optional extra headers) }
//
// When "secret" is set, the request carries an
// "X-CDS-Signature: sha256=<hex hmac>" header computed over the raw request
// body, so the receiving application can verify the call originated from
// this CDS instance.
type appCallbackExecutor struct {
	client *http.Client
}

func (e *appCallbackExecutor) Execute(execCtx *ExecutionContext, action model.Action) error {

	endpoint, _ := action.Config["endpoint_url"].(string)
	endpoint = matcher.ResolveTemplate(endpoint, execCtx.Data)
	if endpoint == "" {
		return fmt.Errorf("app.callback: config.endpoint_url is required")
	}

	payload := buildCallbackPayload(execCtx, action)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("app.callback: failed to serialize payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("app.callback: invalid endpoint_url %q: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")

	if secret, ok := action.Config["secret"].(string); ok && secret != "" {
		req.Header.Set("X-CDS-Signature", "sha256="+signBody(body, secret))
	}
	if headers, ok := action.Config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Header.Set(key, matcher.ResolveTemplate(fmt.Sprintf("%v", value), execCtx.Data))
		}
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("app.callback: request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("app.callback: endpoint %s returned status %d", endpoint, resp.StatusCode)
	}
	return nil
}

func buildCallbackPayload(execCtx *ExecutionContext, action model.Action) map[string]interface{} {

	if custom, ok := action.Config["payload"].(map[string]interface{}); ok && len(custom) > 0 {
		return matcher.ResolveTemplatesInConfig(custom, execCtx.Data)
	}

	payload := map[string]interface{}{
		"rule_id": execCtx.RuleId,
		"event":   execCtx.Event.ToMap(),
	}
	if profile, ok := execCtx.Data["profile"]; ok {
		payload["profile"] = profile
	}
	return payload
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
