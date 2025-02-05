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

// go:embed server.hcl
var serverConfig string

const (
	pathToNomad = "/usr/local/percona/pmm/tools/nomad"
	pathToCerts = "/srv/nomad/certs"
	region      = "global"
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
	err := os.MkdirAll(pathToCerts, 0o755)
	if err != nil {
		return nil, err
	}
	return &Nomad{
		db:     db,
		l:      logrus.WithField("component", "nomad"),
		prefix: "nomad",
	}, nil
}

func (c *Nomad) UpdateConfiguration() error {
	c.m.Lock()
	defer c.m.Unlock()
	settings, err := models.GetSettings(c.db)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}
	if !settings.IsNomadEnabled() {
		c.l.Debugln("Nomad is not enabled")
		return nil
	}
	if settings.PMMPublicAddress == c.cachedPMMAddress {
		c.l.Debugln("Public address is not changed")
		return nil
	}
	c.cachedPMMAddress = settings.PMMPublicAddress
	err = c.generateCACert()
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %w", err)
	}
	err = c.generateServerCert(settings.PMMPublicAddress)
	if err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}
	err = c.generateClientCert()
	if err != nil {
		return fmt.Errorf("failed to generate client certificate: %w", err)
	}
	err = c.generateServerConfig(settings.PMMPublicAddress)
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

	configPath := path.Join(pathToCerts, c.prefix+"-server.hcl")
	configFile, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	m := config{Node: struct{ Address string }{Address: publicAddress}}
	err = tmpl.Execute(configFile, m)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

// generateCACert generates a new CA certificate.
func (c *Nomad) generateCACert() error {
	command := exec.Command(pathToNomad, "ca", "create", "-days", "10000")
	command.Dir = pathToCerts
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) generateServerCert(domain string) error {
	command := exec.Command(pathToNomad,
		"tls",
		"cert",
		"create",
		"-server",
		"-days", "10000",
		"-ca", c.pathToCA(),
		"-key", c.pathToCAKey(),
		"-domain", domain,
		"region", region,
	)
	command.Dir = pathToCerts
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) generateClientCert() error {
	command := exec.Command(pathToNomad,
		"cert",
		"create",
		"-client",
		"-days", "10000",
		"-ca", c.pathToCA(),
		"-key", c.pathToCAKey(),
	)
	command.Dir = pathToCerts
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

func (c *Nomad) GetCACert() (string, error) {
	file, err := os.ReadFile(c.pathToCA())
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func (c *Nomad) GetClientCert() (string, error) {
	file, err := os.ReadFile(path.Join(pathToCerts, "global-client-nomad.pem"))
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func (c *Nomad) GetClientKey() (string, error) {
	file, err := os.ReadFile(path.Join(pathToCerts, "global-client-nomad-key.pem"))
	if err != nil {
		return "", err
	}
	return string(file), nil
}
