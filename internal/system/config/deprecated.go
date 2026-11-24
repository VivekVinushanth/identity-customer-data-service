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

package config

import (
	"fmt"
	"strings"
)

// The endpoint paths of an organization when the deployment file sets none.
const (
	defaultTokenEndpoint         = "/oauth2/token"
	defaultIntrospectionEndpoint = "/oauth2/introspect"
)

// applyDeprecatedKeys moves each key of the deprecated auth_server section to its place under idp
// or auth, where that place is not set, and returns a warning naming the keys it moved. A key that
// is set in both places keeps the new value. It then fills in the default endpoint paths.
func (c *Config) applyDeprecatedKeys() []string {
	old := c.AuthServer
	var moved []string
	move := func(oldKey, newKey, value string, target *string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		moved = append(moved, oldKey+" to "+newKey)
		if strings.TrimSpace(*target) == "" {
			*target = value
		}
	}
	move("host", "idp.host", old.Host, &c.IDP.Host)
	move("port", "idp.port", old.Port, &c.IDP.Port)
	move("tokenEndpoint", "idp.endpoints.token", old.TokenEndpoint, &c.IDP.Endpoints.Token)
	move("introspectionEndpoint", "idp.endpoints.introspection", old.IntrospectionEndPoint,
		&c.IDP.Endpoints.Introspection)
	move("client_id", "idp.client.id", old.ClientID, &c.IDP.Client.ID)
	move("client_secret", "idp.client.secret", old.ClientSecret, &c.IDP.Client.Secret)
	move("introspection_client_id", "idp.introspection_client.id", old.IntrospectionClientID,
		&c.IDP.IntrospectionClient.ID)
	move("introspection_client_secret", "idp.introspection_client.secret", old.IntrospectionClientSecret,
		&c.IDP.IntrospectionClient.Secret)
	move("admin_username", "idp.sync_credentials.username", old.AdminUsername, &c.IDP.SyncCredentials.Username)
	move("admin_password", "idp.sync_credentials.password", old.AdminPassword, &c.IDP.SyncCredentials.Password)
	move("adu_is_hostname", "idp.asgardeo.internal_hostname", old.ADUISHostname,
		&c.IDP.Asgardeo.InternalHostname)
	move("cookieDomain", "auth.cookie_domain", old.CookieDomain, &c.Auth.CookieDomain)
	if len(old.RequiredScopes) > 0 {
		moved = append(moved, "required_scopes to auth.required_scopes")
		if len(c.Auth.RequiredScopes) == 0 {
			c.Auth.RequiredScopes = old.RequiredScopes
		}
	}
	if old.IsSystemAppGrantEnabled {
		moved = append(moved, "the Asgardeo setting to idp.asgardeo.enabled")
		c.IDP.Asgardeo.Enabled = true
	}

	if c.IDP.Endpoints.Token == "" {
		c.IDP.Endpoints.Token = defaultTokenEndpoint
	}
	if c.IDP.Endpoints.Introspection == "" {
		c.IDP.Endpoints.Introspection = defaultIntrospectionEndpoint
	}

	if len(moved) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("The auth_server section is deprecated and will be removed in a later release. "+
		"Move %s in the deployment file. A key set in both places takes the new value.",
		strings.Join(moved, ", "))}
}
