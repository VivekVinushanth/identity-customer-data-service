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

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/wso2/identity-customer-data-service/internal/system/config"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
)

type OrgMgtClient struct {
	BaseURL    string
	PathPrefix string // asgardeo/org-mgt/v2/tenant
	HTTPClient *http.Client
}

func NewOrgMgtClient(cfg config.Config) (*OrgMgtClient, error) {

	baseHostPort := cfg.AuthServer.Host
	if cfg.AuthServer.Port != "" {
		baseHostPort = cfg.AuthServer.Host + ":" + cfg.AuthServer.Port
	}

	httpClient, err := newOutboundHTTPClient(cfg.TLS, cfg.AuthServer.Host)
	if err != nil {
		return nil, err
	}

	return &OrgMgtClient{
		BaseURL:    baseHostPort,
		PathPrefix: strings.Trim(cfg.AuthServer.OrgManagementServiceEndpoint, "/"),
		HTTPClient: httpClient,
	}, nil
}

type OrgDetails map[string]interface{}

func (c *OrgMgtClient) GetOrgByHandle(orgHandle string) (OrgDetails, error) {

	logger := log.GetLogger()
	endpoint := fmt.Sprintf("https://%s/%s/%s", c.BaseURL, c.PathPrefix, orgHandle)
	logger.Info("Obtaining Organization uuid over mTLS private link by communicating with organization management service.")

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        "ORG_MGT_REQUEST_BUILD_FAILED",
			Message:     "Unable to build org-mgt request",
			Description: "Failed to create org-mgt GET request",
		}, err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        "ORG_MGT_CALL_FAILED",
			Message:     "Unable to call org-mgt service",
			Description: "mTLS request to org-mgt failed",
		}, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        "ORG_MGT_NON_200",
			Message:     "Org-mgt returned an error",
			Description: fmt.Sprintf("Status: %d, Response: %s", resp.StatusCode, bodyStr),
		}, fmt.Errorf("org-mgt status %d", resp.StatusCode))
	}

	var out map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        "ORG_MGT_RESPONSE_PARSE_FAILED",
			Message:     "Unable to parse org-mgt response",
			Description: "Invalid JSON returned by org-mgt",
		}, err)
	}

	return out, nil
}
