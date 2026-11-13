// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package nomad

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/vault"
)

// fakePKI issues certificates from a CA of its own, as the PKI secrets engine of Vault does.
type fakePKI struct {
	t      *testing.T
	ca     *x509.Certificate
	caKey  *ecdsa.PrivateKey
	caPEM  string
	ttl    time.Duration
	now    func() time.Time
	m      sync.Mutex
	calls  []vault.CertificateRequest
	paths  []string
	failed error
}

func newFakePKI(t *testing.T, now func() time.Time) *fakePKI {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Vault PKI"},
		NotBefore:             now().Add(-time.Hour),
		NotAfter:              now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &fakePKI{
		t:     t,
		ca:    ca,
		caKey: key,
		caPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		ttl:   3 * time.Hour,
		now:   now,
	}
}

func (p *fakePKI) issue(_ context.Context, path string, req vault.CertificateRequest) (*vault.Certificate, error) {
	p.m.Lock()
	defer p.m.Unlock()

	p.paths = append(p.paths, path)
	p.calls = append(p.calls, req)
	if p.failed != nil {
		return nil, p.failed
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(p.t, err)
	notAfter := p.now().Add(p.ttl)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(int64(len(p.calls) + 1)),
		Subject:      pkix.Name{CommonName: req.CommonName},
		DNSNames:     append([]string{req.CommonName}, req.AltNames...),
		NotBefore:    p.now().Add(-time.Minute),
		NotAfter:     notAfter,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, p.ca, &key.PublicKey, p.caKey)
	require.NoError(p.t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(p.t, err)

	return &vault.Certificate{
		Certificate: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		PrivateKey:  string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})),
		IssuingCA:   p.caPEM,
		Expiration:  notAfter.Truncate(time.Second),
	}, nil
}

func (p *fakePKI) fail(err error) {
	p.m.Lock()
	defer p.m.Unlock()
	p.failed = err
}

func (p *fakePKI) issued() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.calls)
}

// clock is a time that tests move.
type clock struct {
	m sync.Mutex
	t time.Time
}

func (c *clock) now() time.Time {
	c.m.Lock()
	defer c.m.Unlock()
	return c.t
}

func (c *clock) add(d time.Duration) {
	c.m.Lock()
	defer c.m.Unlock()
	c.t = c.t.Add(d)
}

func newVaultNomad(t *testing.T) (*Nomad, *fakePKI, *clock) {
	t.Helper()

	clk := &clock{t: time.Now().Truncate(time.Second)}
	pki := newFakePKI(t, clk.now)
	dir := t.TempDir()
	c := &Nomad{
		l:          logrus.WithField("test", t.Name()),
		prefix:     "nomad",
		certsDir:   dir,
		configPath: filepath.Join(dir, "nomad-server-%s.hcl"),
		pkiPath:    "pki/issue/pmm-nomad",
		issue:      pki.issue,
		now:        clk.now,
		issued:     make(chan struct{}, 1),
	}
	return c, pki, clk
}

func renewAt(c *Nomad) time.Time {
	c.m.Lock()
	defer c.m.Unlock()
	return c.renewAt
}

// testAddress is the public address of PMM Server in the tests.
const testAddress = "pmm.example.com"

func nomadSettings() *models.Settings {
	settings := &models.Settings{PMMPublicAddress: testAddress}
	settings.Nomad.Enabled = new(true)
	return settings
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func TestIssueCertificatesFromVault(t *testing.T) {
	t.Parallel()

	c, pki, clk := newVaultNomad(t)
	// The CA private key that PMM Server kept before the certificates were issued from Vault.
	require.NoError(t, os.WriteFile(c.pathToCAKey(), []byte("old CA key"), 0o600))

	require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))

	assert.Equal(t, []string{"pki/issue/pmm-nomad", "pki/issue/pmm-nomad"}, pki.paths)
	assert.Equal(t, []vault.CertificateRequest{
		{CommonName: "server.global.nomad", AltNames: []string{"localhost"}, IPSANs: []string{"127.0.0.1"}},
		{CommonName: "client.global.nomad", AltNames: []string{"localhost"}, IPSANs: []string{"127.0.0.1"}},
	}, pki.calls)

	assert.Equal(t, pki.caPEM, readFile(t, c.pathToCA()))
	assert.NoFileExists(t, c.pathToCAKey(), "no CA private key is kept")
	for p, perm := range map[string]os.FileMode{
		c.pathToServerCert(testAddress): 0o644,
		c.pathToServerKey(testAddress):  0o600,
		c.pathToClientCert():            0o644,
		c.pathToClientKey():             0o600,
	} {
		fi, err := os.Stat(p)
		require.NoError(t, err)
		assert.Equal(t, perm, fi.Mode().Perm(), p)
	}

	// The agents get the client certificate that was issued.
	clientCert, err := c.GetClientCert()
	require.NoError(t, err)
	block, _ := pem.Decode([]byte(clientCert))
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.Equal(t, "client.global.nomad", cert.Subject.CommonName)
	require.NoError(t, cert.CheckSignatureFrom(pki.ca))

	assert.FileExists(t, filepath.Join(c.certsDir, "nomad-server-"+testAddress+".hcl"))
	assert.Equal(t, clk.now().Add(2*time.Hour), c.renewAt, "two thirds of the lifetime")
	assert.Len(t, c.issued, 1, "the certificates are handed on")
}

func TestIssueCertificatesFromVaultFails(t *testing.T) {
	t.Parallel()

	t.Run("without certificates, Nomad cannot be configured", func(t *testing.T) {
		t.Parallel()

		c, pki, _ := newVaultNomad(t)
		pki.fail(errors.New("permission denied"))

		err := c.UpdateConfiguration(t.Context(), nomadSettings())
		require.ErrorContains(t, err, "failed to issue the certificates of Nomad from Vault: permission denied")
		assert.Empty(t, c.cachedPMMAddress, "tried again with the next update")
	})

	t.Run("valid certificates are kept", func(t *testing.T) {
		t.Parallel()

		c, pki, clk := newVaultNomad(t)
		require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))
		clientCert := readFile(t, c.pathToClientCert())
		<-c.issued

		// As after a restart of PMM Server.
		c.cachedPMMAddress = ""
		clk.add(time.Hour)
		pki.fail(errors.New("connection refused"))
		require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))

		assert.Equal(t, clientCert, readFile(t, c.pathToClientCert()))
		assert.Equal(t, clk.now().Add(renewRetryInterval), c.renewAt, "tried again soon")
		assert.Empty(t, c.issued)
	})

	t.Run("expired certificates are not kept", func(t *testing.T) {
		t.Parallel()

		c, pki, clk := newVaultNomad(t)
		require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))

		c.cachedPMMAddress = ""
		clk.add(4 * time.Hour)
		pki.fail(errors.New("connection refused"))
		err := c.UpdateConfiguration(t.Context(), nomadSettings())
		require.ErrorContains(t, err, "connection refused")
	})
}

func TestRenewCertificates(t *testing.T) {
	t.Parallel()

	t.Run("not from Vault", func(t *testing.T) {
		t.Parallel()

		c := &Nomad{}
		c.RenewCertificates(t.Context(), func(context.Context) { t.Fatal("unexpected") })
	})

	t.Run("issued again when due, and handed on", func(t *testing.T) {
		t.Parallel()

		c, pki, clk := newVaultNomad(t)
		require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))
		firstClientCert := readFile(t, c.pathToClientCert())

		ctx, cancel := context.WithCancel(t.Context())
		handedOn := make(chan struct{}, 10)
		done := make(chan struct{})
		go func() {
			defer close(done)
			c.RenewCertificates(ctx, func(context.Context) { handedOn <- struct{}{} })
		}()

		// The certificates issued by UpdateConfiguration are handed on.
		<-handedOn
		assert.Equal(t, 2, pki.issued())

		// Not issued again before they are due.
		clk.add(time.Hour)
		c.renewIfDue(ctx)
		assert.Equal(t, 2, pki.issued())

		// Called as the timer of the loop calls it, which waits for the real time.
		clk.add(time.Hour)
		c.renewIfDue(ctx)
		<-handedOn

		assert.Equal(t, 4, pki.issued())
		assert.NotEqual(t, firstClientCert, readFile(t, c.pathToClientCert()))
		assert.Equal(t, clk.now().Add(2*time.Hour), renewAt(c))

		cancel()
		<-done
	})

	t.Run("retried when Vault fails", func(t *testing.T) {
		t.Parallel()

		c, pki, clk := newVaultNomad(t)
		require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))
		<-c.issued

		clk.add(2 * time.Hour)
		pki.fail(errors.New("connection refused"))
		c.renewIfDue(t.Context())
		assert.Equal(t, clk.now().Add(renewRetryInterval), c.renewAt)
		assert.Empty(t, c.issued)
	})

	t.Run("nothing is issued while Nomad is disabled", func(t *testing.T) {
		t.Parallel()

		c, pki, clk := newVaultNomad(t)
		require.NoError(t, c.UpdateConfiguration(t.Context(), nomadSettings()))
		<-c.issued

		disabled := &models.Settings{PMMPublicAddress: testAddress}
		disabled.Nomad.Enabled = new(false)
		require.NoError(t, c.UpdateConfiguration(t.Context(), disabled))

		clk.add(3 * time.Hour)
		c.renewIfDue(t.Context())
		assert.Equal(t, 2, pki.issued())
		assert.Equal(t, renewRetryInterval, c.untilRenewal())
	})
}
