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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadDocument(t *testing.T, document string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(document), 0o600); err != nil {
		t.Fatalf("failed to write the deployment file: %v", err)
	}
	cfg, err := LoadConfig(dir, "deployment.yaml")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg
}

// A deployment file of an earlier release, with every key under auth_server, configures the same
// server, and says which keys to move.
func TestLoadConfigMovesTheAuthServerKeys(t *testing.T) {
	cfg := loadDocument(t, `
auth_server:
  host: "is.example.com"
  port: 9443
  adu_is_hostname: "is-internal"
  cookieDomain: "example.com"
  introspectionEndpoint: "/oauth2/introspect"
  tokenEndpoint: "/oauth2/token"
  client_id: "client"
  client_secret: "client-secret"
  introspection_client_id: "introspector"
  introspection_client_secret: "introspector-secret"
  admin_username: "sync"
  admin_password: "sync-secret"
  isSystemAppGrantEnabled: true
  required_scopes:
    profile:view:
      - "internal_cds_profile_view"
`)

	want := IDPConfig{
		Host:                "is.example.com",
		Port:                "9443",
		Endpoints:           IDPEndpointsConfig{Token: "/oauth2/token", Introspection: "/oauth2/introspect"},
		Client:              OAuthClientConfig{ID: "client", Secret: "client-secret"},
		IntrospectionClient: OAuthClientConfig{ID: "introspector", Secret: "introspector-secret"},
		SyncCredentials:     SyncCredentialsConfig{Username: "sync", Password: "sync-secret"},
		Asgardeo:            AsgardeoConfig{Enabled: true, InternalHostname: "is-internal"},
	}
	if cfg.IDP != want {
		t.Errorf("got idp %+v, want %+v", cfg.IDP, want)
	}
	if cfg.Auth.CookieDomain != "example.com" {
		t.Errorf("got auth.cookie_domain %q, want example.com", cfg.Auth.CookieDomain)
	}
	if got := cfg.Auth.RequiredScopes["profile:view"]; len(got) != 1 || got[0] != "internal_cds_profile_view" {
		t.Errorf("got auth.required_scopes %v, want the profile:view scope", cfg.Auth.RequiredScopes)
	}
	warnings := cfg.Deprecations()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "host to idp.host") ||
		!strings.Contains(warnings[0], "required_scopes to auth.required_scopes") {
		t.Errorf("got the warnings %v, want one that names the moved keys", warnings)
	}
}

// A key set in both places keeps the new value.
func TestLoadConfigPrefersTheNewKey(t *testing.T) {
	cfg := loadDocument(t, `
idp:
  host: "new.example.com"
auth_server:
  host: "old.example.com"
`)
	if cfg.IDP.Host != "new.example.com" {
		t.Errorf("got idp.host %q, want new.example.com", cfg.IDP.Host)
	}
	if len(cfg.Deprecations()) != 1 {
		t.Errorf("got the warnings %v, want one for auth_server.host", cfg.Deprecations())
	}
}

// A deployment file without auth_server raises no warning, and gets the default endpoint paths.
func TestLoadConfigWithoutAuthServer(t *testing.T) {
	cfg := loadDocument(t, `
idp:
  type: "wso2is"
  host: "localhost"
`)
	if warnings := cfg.Deprecations(); len(warnings) != 0 {
		t.Errorf("got the warnings %v, want none", warnings)
	}
	if cfg.IDP.Endpoints.Token != "/oauth2/token" || cfg.IDP.Endpoints.Introspection != "/oauth2/introspect" {
		t.Errorf("got the endpoints %+v, want the defaults", cfg.IDP.Endpoints)
	}
	if cfg.IDP.Asgardeo.Enabled {
		t.Error("idp.asgardeo.enabled is on, want it off by default")
	}
}

// The shipped deployment file sets no deprecated key, and configures the identity provider and
// the scopes under their new places.
func TestShippedDeploymentFileSetsNoDeprecatedKey(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", ".."), filepath.Join("config", "repository", "conf",
		"deployment.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if warnings := cfg.Deprecations(); len(warnings) != 0 {
		t.Errorf("got the warnings %v, want none", warnings)
	}
	if cfg.IDP.Host == "" || cfg.IDP.SyncCredentials.Username == "" || len(cfg.Auth.RequiredScopes) == 0 {
		t.Errorf("got idp %+v and %d required scopes, want the host, the sync user and the scopes",
			cfg.IDP, len(cfg.Auth.RequiredScopes))
	}
}
