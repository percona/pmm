// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlshelpers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// customTLSDSN references the TLS config registered by RegisterMySQLCerts.
// Parsing it fails unless that config is present in the driver registry.
const customTLSDSN = "user:pass@tcp(127.0.0.1:3306)/db?tls=custom"

// generateCertPair returns a self-signed certificate and its private key, both PEM-encoded.
func generateCertPair(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "pmm-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	return certPEM, keyPEM
}

// registeredMySQLTLSConfig returns the TLS config the driver resolves for tls=custom.
func registeredMySQLTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	cfg, err := mysql.ParseDSN(customTLSDSN)
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	return cfg.TLS
}

// requireNoMySQLTLSConfig asserts that no TLS config is registered under the custom name.
func requireNoMySQLTLSConfig(t *testing.T) {
	t.Helper()

	_, err := mysql.ParseDSN(customTLSDSN)
	require.ErrorContains(t, err, "unknown config name: custom")
}

// The subtests share the driver's package-level TLS config registry, so they must not run in parallel.
func TestRegisterMySQLCerts(t *testing.T) {
	t.Run("nil files is a no-op", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		require.NoError(t, RegisterMySQLCerts(nil, false))
		requireNoMySQLTLSConfig(t)
	})

	t.Run("empty files still registers a config", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		require.NoError(t, RegisterMySQLCerts(map[string]string{}, true))

		cfg := registeredMySQLTLSConfig(t)
		assert.True(t, cfg.InsecureSkipVerify)
		assert.Empty(t, cfg.Certificates)
		assert.Nil(t, cfg.RootCAs)
	})

	t.Run("valid cert, key and ca", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		cert, key := generateCertPair(t)
		files := map[string]string{
			agentv1.TLSCertFileName: cert,
			agentv1.TLSKeyFileName:  key,
			agentv1.TLSCaFileName:   cert,
		}
		require.NoError(t, RegisterMySQLCerts(files, true))

		cfg := registeredMySQLTLSConfig(t)
		assert.True(t, cfg.InsecureSkipVerify)
		assert.Len(t, cfg.Certificates, 1)
		assert.NotNil(t, cfg.RootCAs)
	})

	t.Run("only ca provided", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		cert, _ := generateCertPair(t)
		files := map[string]string{agentv1.TLSCaFileName: cert}
		require.NoError(t, RegisterMySQLCerts(files, false))

		cfg := registeredMySQLTLSConfig(t)
		assert.False(t, cfg.InsecureSkipVerify)
		assert.Empty(t, cfg.Certificates)
		assert.NotNil(t, cfg.RootCAs)
	})

	t.Run("cert without key skips the client cert", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		cert, _ := generateCertPair(t)
		files := map[string]string{agentv1.TLSCertFileName: cert, agentv1.TLSCaFileName: cert}
		require.NoError(t, RegisterMySQLCerts(files, false))

		cfg := registeredMySQLTLSConfig(t)
		assert.Empty(t, cfg.Certificates)
		assert.NotNil(t, cfg.RootCAs)
	})

	t.Run("invalid cert/key pair", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		files := map[string]string{
			agentv1.TLSCertFileName: "not a cert",
			agentv1.TLSKeyFileName:  "not a key",
		}
		err := RegisterMySQLCerts(files, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "register MySQL client cert failed")
		requireNoMySQLTLSConfig(t)
	})
}

// startValkeyTLSServer starts a TLS listener that only completes handshakes and returns its address.
func startValkeyTLSServer(t *testing.T, cfg *tls.Config) string {
	t.Helper()

	listener, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	ctx := t.Context()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			tlsConn, ok := conn.(*tls.Conn)
			if !ok {
				_ = conn.Close()
				return
			}

			go func() {
				defer tlsConn.Close()
				_ = tlsConn.HandshakeContext(ctx)
			}()
		}
	}()

	return listener.Addr().String()
}

// serverTLSConfig returns a server config presenting the given PEM key pair.
func serverTLSConfig(t *testing.T, certPEM, keyPEM string) *tls.Config {
	t.Helper()

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	require.NoError(t, err)

	return &tls.Config{Certificates: []tls.Certificate{cert}} //nolint:exhaustruct
}

// dialValkey dials the address over the options GetValkeyTLSConfig produced for the given inputs.
func dialValkey(t *testing.T, addr string, files *agentv1.TextFiles, tlsSkipVerify bool) error {
	t.Helper()

	opts, err := GetValkeyTLSConfig(files, true, tlsSkipVerify)
	require.NoError(t, err)

	conn, err := redis.DialURLContext(t.Context(), "rediss://"+addr, opts...)
	if conn != nil {
		t.Cleanup(func() { _ = conn.Close() })
	}

	return err
}

func TestGetValkeyTLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("no TLS returns no options", func(t *testing.T) {
		t.Parallel()
		opts, err := GetValkeyTLSConfig(nil, false, true)
		require.NoError(t, err)
		assert.Nil(t, opts)
	})

	t.Run("nil files still returns dial options", func(t *testing.T) {
		t.Parallel()
		opts, err := GetValkeyTLSConfig(nil, true, true)
		require.NoError(t, err)
		// DialUseTLS, DialTLSSkipVerify and DialTLSConfig.
		assert.Len(t, opts, 3)
	})

	t.Run("empty files still returns dial options", func(t *testing.T) {
		t.Parallel()
		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCertFileName: "",
			agentv1.TLSKeyFileName:  "",
			agentv1.TLSCaFileName:   "",
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("valid files returns dial options", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCertFileName: cert,
			agentv1.TLSKeyFileName:  key,
			agentv1.TLSCaFileName:   cert,
		}}
		opts, err := GetValkeyTLSConfig(files, true, true)
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("ca only returns dial options", func(t *testing.T) {
		t.Parallel()
		cert, _ := generateCertPair(t)
		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCaFileName: cert,
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("client key pair without ca returns dial options", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCertFileName: cert,
			agentv1.TLSKeyFileName:  key,
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("half a client key pair is skipped", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)

		for name, files := range map[string]map[string]string{
			"cert without key": {agentv1.TLSCertFileName: cert},
			"key without cert": {agentv1.TLSKeyFileName: key},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				opts, err := GetValkeyTLSConfig(&agentv1.TextFiles{Files: files}, true, true)
				require.NoError(t, err)
				assert.Len(t, opts, 3)
			})
		}
	})

	t.Run("invalid cert/key pair returns error", func(t *testing.T) {
		t.Parallel()
		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCertFileName: "bad",
			agentv1.TLSKeyFileName:  "bad",
			agentv1.TLSCaFileName:   "bad",
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.Error(t, err)
		assert.Nil(t, opts)
	})

	t.Run("unparseable ca returns error", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCertFileName: cert,
			agentv1.TLSKeyFileName:  key,
			agentv1.TLSCaFileName:   "not a valid ca",
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.ErrorContains(t, err, "failed to append certs from PEM")
		assert.Nil(t, opts)
	})
}

// The options only take effect inside redigo, so the cases that matter are asserted on a real handshake.
func TestGetValkeyTLSConfigHandshake(t *testing.T) {
	t.Parallel()

	t.Run("skip verify without certificates accepts an untrusted server", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		addr := startValkeyTLSServer(t, serverTLSConfig(t, cert, key))

		require.NoError(t, dialValkey(t, addr, nil, true))
	})

	t.Run("without skip verify an untrusted server is rejected", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		addr := startValkeyTLSServer(t, serverTLSConfig(t, cert, key))

		err := dialValkey(t, addr, nil, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})

	t.Run("certificate authority alone verifies the server", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		addr := startValkeyTLSServer(t, serverTLSConfig(t, cert, key))
		files := &agentv1.TextFiles{Files: map[string]string{agentv1.TLSCaFileName: cert}}

		require.NoError(t, dialValkey(t, addr, files, false))
	})

	t.Run("client key pair satisfies a server requiring mutual TLS", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)

		clientCAs := x509.NewCertPool()
		require.True(t, clientCAs.AppendCertsFromPEM([]byte(cert)))

		cfg := serverTLSConfig(t, cert, key)
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		cfg.ClientCAs = clientCAs
		addr := startValkeyTLSServer(t, cfg)

		files := &agentv1.TextFiles{Files: map[string]string{
			agentv1.TLSCertFileName: cert,
			agentv1.TLSKeyFileName:  key,
			agentv1.TLSCaFileName:   cert,
		}}
		require.NoError(t, dialValkey(t, addr, files, false))
	})
}
