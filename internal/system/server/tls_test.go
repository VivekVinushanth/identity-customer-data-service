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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/identity-customer-data-service/internal/system/config"
)

// newClientCertificate returns a self-signed key pair for TLS client authentication, and its certificate in PEM.
func newClientCertificate(t *testing.T, name string) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestServerTLSConfigWithoutRequireClientCertAsksForNoClientCertificate(t *testing.T) {
	tlsConfig, err := serverTLSConfig(config.TLSConfig{}, t.TempDir())

	require.NoError(t, err)
	assert.Equal(t, tls.NoClientCert, tlsConfig.ClientAuth)
	assert.Nil(t, tlsConfig.ClientCAs)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
}

func TestServerTLSConfigRefusesRequireClientCertWithoutItsOwnTrustStore(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.pem"), []byte("not a certificate"), 0o600))

	for name, cfg := range map[string]config.TLSConfig{
		"no client trust store":        {RequireClientCert: true, TrustStore: "trust_store.pem"},
		"a client trust store missing": {RequireClientCert: true, ClientTrustStore: "missing.pem"},
		"a client trust store empty":   {RequireClientCert: true, ClientTrustStore: "bad.pem"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := serverTLSConfig(cfg, dir)

			assert.Error(t, err)
		})
	}
}

// TestServerTLSConfigVerifiesAPresentedCertificateAndLetsOthersThrough runs real handshakes against the
// listener's TLS: browsers come without a certificate, the identity server with a trusted one, and a
// certificate the client trust store does not name ends the handshake.
func TestServerTLSConfigVerifiesAPresentedCertificateAndLetsOthersThrough(t *testing.T) {
	trusted, trustedPEM := newClientCertificate(t, "identity-server")
	untrusted, _ := newClientCertificate(t, "someone-else")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client_trust_store.pem"), trustedPEM, 0o600))

	tlsConfig, err := serverTLSConfig(config.TLSConfig{RequireClientCert: true, ClientTrustStore: "client_trust_store.pem"},
		dir)
	require.NoError(t, err)
	assert.Equal(t, tls.VerifyClientCertIfGiven, tlsConfig.ClientAuth)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.VerifiedChains) > 0 {
			_, _ = w.Write([]byte(r.TLS.VerifiedChains[0][0].Subject.CommonName))
			return
		}
		_, _ = w.Write([]byte("anonymous"))
	}))
	server.TLS = tlsConfig
	server.StartTLS()
	t.Cleanup(server.Close)

	// Sends the certificate whatever CAs the server names, as a Java client does: a Go client left to choose
	// would hold back one the server's CA list does not name.
	call := func(certificates ...tls.Certificate) (string, error) {
		transport := server.Client().Transport.(*http.Transport).Clone()
		transport.TLSClientConfig.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			if len(certificates) == 0 {
				return &tls.Certificate{}, nil
			}
			return &certificates[0], nil
		}
		resp, err := (&http.Client{Transport: transport}).Get(server.URL)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 64)
		n, _ := resp.Body.Read(body)
		return string(body[:n]), nil
	}

	caller, err := call()
	require.NoError(t, err, "a caller without a certificate is let through")
	assert.Equal(t, "anonymous", caller)

	caller, err = call(trusted)
	require.NoError(t, err)
	assert.Equal(t, "identity-server", caller, "the trusted certificate is verified")

	_, err = call(untrusted)
	assert.Error(t, err, "a certificate the client trust store does not name ends the handshake")
}
