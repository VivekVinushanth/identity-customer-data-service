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
	"strings"

	"github.com/wso2/identity-customer-data-service/internal/system/certs"
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
		if len(tlsCfg.ClientCertNames) > 0 {
			return nil, errors.New("tls.client_cert_names is set but tls.require_client_cert is off")
		}
		return tlsConfig, nil
	}
	for _, name := range tlsCfg.ClientCertNames {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("tls.client_cert_names holds an empty name")
		}
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
	if err := checkDefaultCertificateAlone(tlsCfg, certDir, trustPath, trustPEM); err != nil {
		return nil, err
	}

	tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	tlsConfig.ClientCAs = clientCAs
	return tlsConfig, nil
}

// checkDefaultCertificateAlone refuses a client trust store that still trusts the default certificate the pack
// ships with once the deployment has certificates of its own: a client trust store that holds other certificates
// as well, or a server certificate that is not the default. The private key of the default is in every Identity
// Server 7.3.0 download, so anyone could present it on the sync endpoints, which an operator who replaced the
// certificates does not expect. A pack that keeps every default, as a first installation does, starts as before.
func checkDefaultCertificateAlone(tlsCfg config.TLSConfig, certDir, trustPath string, trustPEM []byte) error {
	trustsDefault, trustsOthers := false, false
	for _, cert := range certs.ParsePEM(trustPEM) {
		if certs.IsDefault(cert) {
			trustsDefault = true
		} else {
			trustsOthers = true
		}
	}
	if !trustsDefault {
		return nil
	}
	if trustsOthers {
		return fmt.Errorf("client_trust_store %s trusts the default certificate the pack ships with besides "+
			"certificates of this deployment. Its private key is in every Identity Server 7.3.0 download: remove it",
			trustPath)
	}
	serverCerts, err := certs.ReadPEM(filepath.Join(certDir, tlsCfg.CDSPublicCert))
	if err != nil || len(serverCerts) == 0 || certs.IsDefault(serverCerts[0]) {
		return nil
	}
	return fmt.Errorf("client_trust_store %s trusts the default certificate the pack ships with, while CDD "+
		"presents a certificate of its own. Its private key is in every Identity Server 7.3.0 download: replace it "+
		"with the certificate the Identity Server presents to CDD", trustPath)
}
