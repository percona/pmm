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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

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

func TestRegisterMySQLCerts(t *testing.T) {
	t.Parallel()

	t.Run("nil files is a no-op", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, RegisterMySQLCerts(nil, false))
	})

	t.Run("valid cert, key and ca", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		files := map[string]string{
			"tlsCert": cert,
			"tlsKey":  key,
			"tlsCa":   cert,
		}
		err := RegisterMySQLCerts(files, true)
		require.NoError(t, err)
		DeregisterMySQLCerts()
	})

	t.Run("only ca provided", func(t *testing.T) {
		t.Parallel()
		cert, _ := generateCertPair(t)
		files := map[string]string{"tlsCa": cert}
		err := RegisterMySQLCerts(files, false)
		require.NoError(t, err)
		DeregisterMySQLCerts()
	})

	t.Run("invalid cert/key pair", func(t *testing.T) {
		t.Parallel()
		files := map[string]string{
			"tlsCert": "not a cert",
			"tlsKey":  "not a key",
		}
		err := RegisterMySQLCerts(files, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "register MySQL client cert failed")
	})
}

func TestGetValkeyTLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil files returns no options", func(t *testing.T) {
		t.Parallel()
		opts, err := GetValkeyTLSConfig(nil, true, false)
		require.NoError(t, err)
		assert.Nil(t, opts)
	})

	t.Run("empty files returns no options", func(t *testing.T) {
		t.Parallel()
		files := &agentv1.TextFiles{Files: map[string]string{
			"tlsCert": "",
			"tlsKey":  "",
			"tlsCa":   "",
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.NoError(t, err)
		assert.Nil(t, opts)
	})

	t.Run("valid files returns dial options", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		files := &agentv1.TextFiles{Files: map[string]string{
			"tlsCert": cert,
			"tlsKey":  key,
			"tlsCa":   cert,
		}}
		opts, err := GetValkeyTLSConfig(files, true, true)
		require.NoError(t, err)
		// DialUseTLS, DialTLSSkipVerify and DialTLSConfig.
		assert.Len(t, opts, 3)
	})

	t.Run("invalid cert/key pair returns error", func(t *testing.T) {
		t.Parallel()
		files := &agentv1.TextFiles{Files: map[string]string{
			"tlsCert": "bad",
			"tlsKey":  "bad",
			"tlsCa":   "bad",
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.Error(t, err)
		assert.Nil(t, opts)
	})

	t.Run("unparseable ca returns error", func(t *testing.T) {
		t.Parallel()
		cert, key := generateCertPair(t)
		files := &agentv1.TextFiles{Files: map[string]string{
			"tlsCert": cert,
			"tlsKey":  key,
			"tlsCa":   "not a valid ca",
		}}
		opts, err := GetValkeyTLSConfig(files, true, false)
		require.ErrorContains(t, err, "failed to append certs from PEM")
		assert.Nil(t, opts)
	})
}

func TestIsEmptyTLSFiles(t *testing.T) {
	t.Parallel()

	cert, key := generateCertPair(t)

	tests := []struct {
		name  string
		files *agentv1.TextFiles
		empty bool
	}{
		{name: "nil", files: nil, empty: true},
		{name: "nil map", files: &agentv1.TextFiles{}, empty: true},
		{name: "empty map", files: &agentv1.TextFiles{Files: map[string]string{}}, empty: true},
		{
			name:  "all blank values",
			files: &agentv1.TextFiles{Files: map[string]string{"tlsCert": "", "tlsKey": "", "tlsCa": ""}},
			empty: true,
		},
		{
			name:  "cert and key set",
			files: &agentv1.TextFiles{Files: map[string]string{"tlsCert": cert, "tlsKey": key}},
			empty: false,
		},
		{
			name:  "only ca set",
			files: &agentv1.TextFiles{Files: map[string]string{"tlsCa": cert}},
			empty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.empty, isEmptyTLSFiles(tt.files))
		})
	}
}
