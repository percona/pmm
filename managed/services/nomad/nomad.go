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
	_ "embed" // embed is used to embed server.hcl file.
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"text/template"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

//go:embed server.hcl
var serverConfig string

const (
	pathToNomad       = "/usr/local/percona/pmm/tools/nomad"
	pathToCerts       = "/srv/nomad/certs"
	pathToNomadConfig = "/srv/nomad/nomad-server-%s.hcl"
	region            = "global"
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
}

// New creates a new Nomad client.
func New(db *reform.DB) (*Nomad, error) {
	err := os.MkdirAll(pathToCerts, 0o750) //nolint:mnd
	if err != nil {
		return nil, err
	}
	return &Nomad{
		db:     db,
		l:      logrus.WithField("component", "nomad"),
		prefix: "nomad",
	}, nil
}

// UpdateConfiguration retrieves and applies updated settings for Nomad, regenerates certificates, and updates server config.
func (c *Nomad) UpdateConfiguration(settings *models.Settings) error {
	c.m.Lock()
	defer c.m.Unlock()
	if !settings.IsNomadEnabled() {
		c.l.Debugln("Nomad is not enabled")
		return nil
	}
	address := settings.PMMPublicAddress
	if address == c.cachedPMMAddress {
		c.l.Debugln("Public address is not changed")
		return nil
	}
	c.cachedPMMAddress = address
	var err error
	err = c.generateCACert()
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %w", err)
	}
	err = c.generateServerCert(address)
	if err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}
	err = c.generateClientCert()
	if err != nil {
		return fmt.Errorf("failed to generate client certificate: %w", err)
	}
	err = c.generateServerConfig(address)
	if err != nil {
		return fmt.Errorf("failed to generate server config: %w", err)
	}
	return nil
}

func (c *Nomad) generateServerConfig(publicAddress string) error {
	tmpl, err := template.New("nomad-server.hcl").Parse(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	configPath := path.Join(fmt.Sprintf(pathToNomadConfig, publicAddress))
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

// generateCACert generates a new CA certificate.
func (c *Nomad) generateCACert() error {
	filePaths := []string{
		c.pathToCA(),
		c.pathToCAKey(),
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

	command := exec.Command(pathToNomad, "tls", "ca", "create", "-days", "10000")
	command.Dir = pathToCerts
	command.Stderr = c.l.WriterLevel(logrus.ErrorLevel)
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) generateServerCert(domain string) error {
	filePaths := []string{
		path.Join(pathToCerts, region+"-server-"+domain+".pem"),
		path.Join(pathToCerts, region+"-server-"+domain+"-key.pem"),
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
		"-region", region,
	)
	command.Dir = pathToCerts
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
		"-region", region,
	)
	command.Dir = pathToCerts
	command.Stderr = c.l.WriterLevel(logrus.ErrorLevel)
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) pathToCA() string {
	return path.Join(pathToCerts, c.prefix+"-agent-ca.pem")
}

func (c *Nomad) pathToCAKey() string {
	return path.Join(pathToCerts, c.prefix+"-agent-ca-key.pem")
}

func (c *Nomad) pathToClientKey() string {
	return path.Join(pathToCerts, "global-client-nomad-key.pem")
}

func (c *Nomad) pathToClientCert() string {
	return path.Join(pathToCerts, "global-client-nomad.pem")
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
