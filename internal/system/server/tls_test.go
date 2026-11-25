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
	return newCertificate(t, name, x509.ExtKeyUsageClientAuth)
}

// newCertificate returns a self-signed key pair with the given extended key usages, and its certificate in PEM.
func newCertificate(t *testing.T, name string, usages ...x509.ExtKeyUsage) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  usages,
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
	serverOnly, serverOnlyPEM := newCertificate(t, "bought-from-a-public-ca", x509.ExtKeyUsageServerAuth)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client_trust_store.pem"), append(trustedPEM, serverOnlyPEM...),
		0o600))

	tlsConfig, err := serverTLSConfig(config.TLSConfig{RequireClientCert: true,
		ClientTrustStore: "client_trust_store.pem"}, dir)
	require.NoError(t, err)
	assert.Equal(t, tls.VerifyClientCertIfGiven, tlsConfig.ClientAuth)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := "anonymous"
		if len(r.TLS.VerifiedChains) > 0 {
			caller = r.TLS.VerifiedChains[0][0].Subject.CommonName
		}
		w.Header().Set("X-Caller", caller)
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
		return resp.Header.Get("X-Caller"), nil
	}

	caller, err := call()
	require.NoError(t, err, "a caller without a certificate is let through")
	assert.Equal(t, "anonymous", caller)

	caller, err = call(trusted)
	require.NoError(t, err)
	assert.Equal(t, "identity-server", caller, "the trusted certificate is verified")

	_, err = call(untrusted)
	assert.Error(t, err, "a certificate the client trust store does not name ends the handshake")

	// Why the identity server needs a client certificate of its own once its server certificate comes from a
	// public CA: one for servers only ends the handshake even when the client trust store names it.
	_, err = call(serverOnly)
	assert.ErrorContains(t, err, "bad certificate")
}

// defaultCertificatePEM is the certificate the pack ships: Identity Server 7.3.0's default, public part only.
func defaultCertificatePEM(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "cli", "testdata", "is-7.3.0-default.crt"))
	require.NoError(t, err)
	return raw
}

// The default certificate's private key is in every Identity Server download. A pack that keeps every default
// starts as before; one whose certificates were replaced must not still trust the default on the sync endpoints.
func TestServerTLSConfigTrustsTheDefaultCertificateOnlyWhileEverythingIsDefault(t *testing.T) {
	defaultPEM := defaultCertificatePEM(t)
	_, ownPEM := newClientCertificate(t, "is.example.com")

	for name, tc := range map[string]struct {
		clientTrustStore []byte
		serverCert       []byte
		refused          bool
	}{
		"every default, as a first installation": {clientTrustStore: defaultPEM, serverCert: defaultPEM},
		"replaced on both sides":                 {clientTrustStore: ownPEM, serverCert: ownPEM},
		"the client trust store replaced only":   {clientTrustStore: ownPEM, serverCert: defaultPEM},
		"the default besides a certificate of the deployment": {
			clientTrustStore: append(append([]byte{}, ownPEM...), defaultPEM...), serverCert: defaultPEM, refused: true,
		},
		"the default while CDD presents its own certificate": {
			clientTrustStore: defaultPEM, serverCert: ownPEM, refused: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "client_trust_store.pem"), tc.clientTrustStore, 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "server.crt"), tc.serverCert, 0o600))

			_, err := serverTLSConfig(config.TLSConfig{RequireClientCert: true, CDSPublicCert: "server.crt",
				ClientTrustStore: "client_trust_store.pem"}, dir)

			if tc.refused {
				assert.ErrorContains(t, err, "trusts the default certificate the pack ships with")
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestServerTLSConfigChecksClientCertNames(t *testing.T) {
	dir := t.TempDir()
	_, trustedPEM := newClientCertificate(t, "identity-server")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client_trust_store.pem"), trustedPEM, 0o600))

	_, err := serverTLSConfig(config.TLSConfig{ClientCertNames: []string{"is.example.com"}}, dir)
	assert.ErrorContains(t, err, "tls.require_client_cert is off")

	_, err = serverTLSConfig(config.TLSConfig{RequireClientCert: true, ClientTrustStore: "client_trust_store.pem",
		ClientCertNames: []string{"is.example.com", " "}}, dir)
	assert.ErrorContains(t, err, "empty name")

	_, err = serverTLSConfig(config.TLSConfig{RequireClientCert: true, ClientTrustStore: "client_trust_store.pem",
		ClientCertNames: []string{"is.example.com"}}, dir)
	assert.NoError(t, err)
}
