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

// Package nomad provides implementation for Nomad operations.
package nomad

import (
	"context"
	"crypto/x509"
	_ "embed" // embed is used to embed server.hcl file.
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

//go:embed server.hcl
var serverConfig string

const (
	pathToNomad       = "/usr/local/percona/pmm/tools/nomad"
	pathToCerts       = "/srv/nomad/certs"
	pathToNomadConfig = "/srv/nomad/nomad-server-%s.hcl"
	region            = "global"

	// PKIPathEnvVar names the issue endpoint, <mount>/issue/<role>, of a role of a Vault PKI secrets
	// engine that the certificates of Nomad are issued from. Without it, PMM Server issues them from a
	// CA of its own, whose private key it keeps with them.
	PKIPathEnvVar = "PMM_NOMAD_VAULT_PKI_PATH"

	// How long to wait after the certificates could not be issued from Vault, before trying again.
	renewRetryInterval = time.Minute
)

type config struct {
	Node struct {
		Address string
	}
}

// Nomad is a wrapper around Nomad client.
type Nomad struct {
	db *reform.DB
	l  *logrus.Entry

	m sync.Mutex

	prefix           string
	cachedPMMAddress string
	clientConfig     models.NomadClient

	certsDir string
	// configPath is the path of the configuration file of the Nomad server, a format with the public address.
	configPath string
	pkiPath    string
	issue      func(ctx context.Context, path string, req vault.CertificateRequest) (*vault.Certificate, error)
	now        func() time.Time
	// renewAt is when the certificates issued from Vault are to be issued again; zero until they are issued.
	renewAt time.Time
	// issued is signaled when certificates are issued from Vault, so that RenewCertificates hands them on.
	issued chan struct{}
}

// New creates a new Nomad client.
func New(db *reform.DB, clientConfig *models.NomadClient) (*Nomad, error) {
	err := os.MkdirAll(pathToCerts, 0o750) //nolint:mnd
	if err != nil {
		return nil, err
	}
	return &Nomad{
		db:           db,
		l:            logrus.WithField("component", "nomad"),
		prefix:       "nomad",
		clientConfig: *clientConfig,
		certsDir:     pathToCerts,
		configPath:   pathToNomadConfig,
		pkiPath:      os.Getenv(PKIPathEnvVar),
		issue:        secrets.IssueCertificate,
		now:          time.Now,
		issued:       make(chan struct{}, 1),
	}, nil
}

// UpdateConfiguration retrieves and applies updated settings for Nomad, regenerates certificates, and updates server config.
func (c *Nomad) UpdateConfiguration(ctx context.Context, settings *models.Settings) error {
	c.m.Lock()
	defer c.m.Unlock()
	if !settings.IsNomadEnabled() {
		c.l.Debugln("Nomad is not enabled")
		// Enabled again, it gets certificates again: those it had may have expired since.
		c.cachedPMMAddress = ""
		c.renewAt = time.Time{}
		return nil
	}
	address := settings.PMMPublicAddress
	if address == c.cachedPMMAddress {
		c.l.Debugln("Public address is not changed")
		return nil
	}

	var err error
	if c.pkiPath != "" {
		err = c.issueCertificatesOrKeep(ctx, address)
		if err != nil {
			return err
		}
	} else {
		err = c.generateCertificates(address)
		if err != nil {
			return err
		}
	}
	err = c.generateServerConfig(address)
	if err != nil {
		return fmt.Errorf("failed to generate server config: %w", err)
	}
	c.cachedPMMAddress = address
	return nil
}

// generateCertificates generates the certificates of Nomad from the CA of PMM Server's own, which it creates first
// if there is none.
func (c *Nomad) generateCertificates(address string) error {
	created, err := c.generateCACert()
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %w", err)
	}
	if created {
		// A client certificate kept from before was issued by another CA, such as the one of Vault.
		err = removeFiles(c.pathToClientCert(), c.pathToClientKey())
		if err != nil {
			return fmt.Errorf("failed to remove client certificate: %w", err)
		}
	}
	err = c.generateServerCert(address)
	if err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}
	err = c.generateClientCert()
	if err != nil {
		return fmt.Errorf("failed to generate client certificate: %w", err)
	}
	return nil
}

// issueCertificatesOrKeep issues the certificates of Nomad from Vault. If Vault cannot issue them, the certificates
// issued before are kept while they are valid, and RenewCertificates tries again: Nomad keeps working meanwhile.
// Must be called with c.m held.
func (c *Nomad) issueCertificatesOrKeep(ctx context.Context, address string) error {
	err := c.issueCertificates(ctx, address)
	if err == nil {
		return nil
	}

	validUntil, validErr := c.issuedCertificatesValidUntil(address)
	if validErr != nil || !c.now().Before(validUntil) {
		return fmt.Errorf("failed to issue the certificates of Nomad from Vault: %w", err)
	}
	c.l.Warnf("Failed to issue the certificates of Nomad from Vault, keeping those valid until %s: %s.", validUntil, err)
	c.renewAt = c.now().Add(renewRetryInterval)
	return nil
}

// issueCertificates issues the server and client certificates of Nomad from the role of the Vault PKI secrets
// engine that c.pkiPath names, and removes the CA private key of PMM Server's own that may be left from before.
// Must be called with c.m held.
func (c *Nomad) issueCertificates(ctx context.Context, address string) error {
	issuedAt := c.now()
	// The names that "nomad tls cert create" gives them: Nomad verifies the name server.<region>.nomad.
	server, err := c.issue(ctx, c.pkiPath, vault.CertificateRequest{
		CommonName: "server." + region + ".nomad",
		AltNames:   []string{"localhost"},
		IPSANs:     []string{"127.0.0.1"},
	})
	if err != nil {
		return err
	}
	client, err := c.issue(ctx, c.pkiPath, vault.CertificateRequest{
		CommonName: "client." + region + ".nomad",
		AltNames:   []string{"localhost"},
		IPSANs:     []string{"127.0.0.1"},
	})
	if err != nil {
		return err
	}

	for _, f := range []struct {
		path    string
		content string
		perm    os.FileMode
	}{
		{c.pathToCA(), server.IssuingCA, 0o644},
		{c.pathToServerCert(address), server.Certificate, 0o644},
		{c.pathToServerKey(address), server.PrivateKey, 0o600},
		{c.pathToClientCert(), client.Certificate, 0o644},
		{c.pathToClientKey(), client.PrivateKey, 0o600},
	} {
		err = writeFile(f.path, f.content, f.perm)
		if err != nil {
			return err
		}
	}
	err = removeFiles(c.pathToCAKey())
	if err != nil {
		return fmt.Errorf("failed to remove the CA private key: %w", err)
	}

	// Issued again once two thirds of the lifetime of the one that expires first have passed.
	expiration := server.Expiration
	if client.Expiration.Before(expiration) {
		expiration = client.Expiration
	}
	c.renewAt = issuedAt.Add(expiration.Sub(issuedAt) * 2 / 3) //nolint:mnd
	c.l.Infof("Issued the certificates of Nomad from Vault, valid until %s.", expiration)

	select {
	case c.issued <- struct{}{}:
	default:
	}
	return nil
}

// issuedCertificatesValidUntil returns when the first of the server and client certificates of Nomad expires.
func (c *Nomad) issuedCertificatesValidUntil(address string) (time.Time, error) {
	var validUntil time.Time
	for _, p := range []string{c.pathToServerCert(address), c.pathToClientCert()} {
		b, err := os.ReadFile(p) //nolint:gosec
		if err != nil {
			return time.Time{}, err
		}
		block, _ := pem.Decode(b)
		if block == nil {
			return time.Time{}, fmt.Errorf("%s: no PEM data", p)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return time.Time{}, fmt.Errorf("%s: %w", p, err)
		}
		if validUntil.IsZero() || cert.NotAfter.Before(validUntil) {
			validUntil = cert.NotAfter
		}
	}
	return validUntil, nil
}

// RenewCertificates issues the certificates of Nomad from Vault again before they expire, until ctx is done.
// It calls issued after certificates were issued, so that the Nomad server and agents take them. It returns
// at once if the certificates of Nomad are not issued from Vault.
func (c *Nomad) RenewCertificates(ctx context.Context, issued func(ctx context.Context)) {
	if c.pkiPath == "" {
		return
	}

	t := time.NewTimer(c.untilRenewal())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.issued:
			issued(ctx)
		case <-t.C:
			c.renewIfDue(ctx)
		}
		t.Reset(c.untilRenewal())
	}
}

// untilRenewal returns how long to wait before checking whether the certificates are due to be issued again.
func (c *Nomad) untilRenewal() time.Duration {
	c.m.Lock()
	defer c.m.Unlock()

	if c.renewAt.IsZero() {
		// Nothing is issued while Nomad is disabled.
		return renewRetryInterval
	}
	return max(c.renewAt.Sub(c.now()), 0)
}

// renewIfDue issues the certificates of Nomad again if they are due to be, signaling c.issued once it did.
func (c *Nomad) renewIfDue(ctx context.Context) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.renewAt.IsZero() || c.now().Before(c.renewAt) || c.cachedPMMAddress == "" {
		return
	}
	err := c.issueCertificates(ctx, c.cachedPMMAddress)
	if err != nil {
		c.l.Errorf("Failed to issue the certificates of Nomad from Vault again, retrying in %s: %s.", renewRetryInterval, err)
		c.renewAt = c.now().Add(renewRetryInterval)
	}
}

func (c *Nomad) generateServerConfig(publicAddress string) error {
	tmpl, err := template.New("nomad-server.hcl").Parse(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	configPath := path.Join(fmt.Sprintf(c.configPath, publicAddress))
	configFile, err := os.Create(configPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close() //nolint:errcheck

	m := config{Node: struct{ Address string }{Address: publicAddress}}
	err = tmpl.Execute(configFile, m)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

// generateCACert generates a new CA certificate unless there is one with its private key, and reports whether it did.
// The private key is missing after the certificates were issued from Vault.
func (c *Nomad) generateCACert() (bool, error) {
	filePaths := []string{
		c.pathToCA(),
		c.pathToCAKey(),
	}
	present := 0
	for _, filePath := range filePaths {
		statInfo, err := os.Stat(filePath)
		if err == nil && statInfo.Size() > 0 {
			present++
		}
	}
	if present == len(filePaths) {
		return false, nil
	}
	err := removeFiles(filePaths...)
	if err != nil {
		return false, fmt.Errorf("failed to remove CA certificate: %w", err)
	}

	command := exec.Command(pathToNomad, "tls", "ca", "create", "-days", "10000")
	command.Dir = c.certsDir
	command.Stderr = c.l.WriterLevel(logrus.ErrorLevel)
	err = command.Run()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Nomad) generateServerCert(domain string) error {
	filePaths := []string{
		c.pathToServerCert(domain),
		c.pathToServerKey(domain),
	}
	for _, filePath := range filePaths {
		statInfo, err := os.Stat(filePath)
		if (err == nil && statInfo.Size() > 0) || !os.IsNotExist(err) {
			err := os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove Server certificate: %w", err)
			}
		}
	}
	command := exec.Command(pathToNomad, //nolint:gosec
		"tls",
		"cert",
		"create",
		"-server",
		"-days", "10000",
		"-ca", c.pathToCA(),
		"-key", c.pathToCAKey(),
		"-domain", domain,
		"-additional-dnsname", "server."+region+".nomad",
		"-region", region)
	command.Dir = c.certsDir
	command.Stderr = c.l.WriterLevel(logrus.ErrorLevel)
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) generateClientCert() error {
	filePaths := []string{
		c.pathToClientCert(),
		c.pathToClientKey(),
	}
	for _, filePath := range filePaths {
		statInfo, err := os.Stat(filePath)
		if err == nil && statInfo.Size() > 0 {
			return nil
		} else if !os.IsNotExist(err) {
			err := os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove CA certificate: %w", err)
			}
		}
	}
	command := exec.Command(pathToNomad, //nolint:gosec
		"tls",
		"cert",
		"create",
		"-client",
		"-days", "10000",
		"-ca", c.pathToCA(),
		"-key", c.pathToCAKey(),
		"-region", region)
	command.Dir = c.certsDir
	command.Stderr = c.l.WriterLevel(logrus.ErrorLevel)
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) pathToCA() string {
	return path.Join(c.certsDir, c.prefix+"-agent-ca.pem")
}

func (c *Nomad) pathToCAKey() string {
	return path.Join(c.certsDir, c.prefix+"-agent-ca-key.pem")
}

func (c *Nomad) pathToServerCert(address string) string {
	return path.Join(c.certsDir, region+"-server-"+address+".pem")
}

func (c *Nomad) pathToServerKey(address string) string {
	return path.Join(c.certsDir, region+"-server-"+address+"-key.pem")
}

func (c *Nomad) pathToClientKey() string {
	return path.Join(c.certsDir, "global-client-nomad-key.pem")
}

func (c *Nomad) pathToClientCert() string {
	return path.Join(c.certsDir, "global-client-nomad.pem")
}

// writeFile replaces the file at path with one holding content, so that a reader never sees it half written.
func writeFile(path, content string, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	_, err = tmp.WriteString(content)
	if err == nil {
		err = tmp.Chmod(perm)
	}
	if err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// removeFiles removes the files that exist of the given ones.
func removeFiles(paths ...string) error {
	var errs []error
	for _, p := range paths {
		err := os.Remove(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// GetCACert reads and returns the content of the CA certificate file for Nomad. Returns an error if it fails.
func (c *Nomad) GetCACert() (string, error) {
	file, err := os.ReadFile(c.pathToCA())
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// GetClientCert reads and returns the content of the global client certificate file for Nomad. Returns an error if it fails.
func (c *Nomad) GetClientCert() (string, error) {
	file, err := os.ReadFile(c.pathToClientCert())
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// GetClientKey reads and returns the content of the global client key file for Nomad. Returns an error if it fails.
func (c *Nomad) GetClientKey() (string, error) {
	file, err := os.ReadFile(c.pathToClientKey())
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// GetClientConfig returns the Nomad client configuration.
func (c *Nomad) GetClientConfig() models.NomadClient {
	return c.clientConfig
}
