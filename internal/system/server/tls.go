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

package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wso2/identity-customer-data-service/internal/system/config"
)

// serverTLSConfig is the TLS the listener serves.
//
// With require_client_cert on, a client certificate is verified against the client
// trust store when one is presented, and a caller without one is let through:
// browsers and SDKs reach the same port. The sync endpoints the identity server
// calls are the ones that refuse a request without a verified certificate.
func serverTLSConfig(tlsCfg config.TLSConfig, certDir string) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if !tlsCfg.RequireClientCert {
		return tlsConfig, nil
	}

	// Never the system CAs: any certificate a public CA issued would pass.
	if tlsCfg.ClientTrustStore == "" {
		return nil, errors.New("tls.require_client_cert is on but tls.client_trust_store is not set")
	}
	trustPath := filepath.Join(certDir, tlsCfg.ClientTrustStore)
	trustPEM, err := os.ReadFile(filepath.Clean(trustPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read client_trust_store at %s: %w", trustPath, err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(trustPEM) {
		return nil, fmt.Errorf("no certificates in client_trust_store: %s", trustPath)
	}

	tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	tlsConfig.ClientCAs = clientCAs
	return tlsConfig, nil
}
