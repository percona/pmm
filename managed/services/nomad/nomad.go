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
	"os"
	"os/exec"
	"path"
)

// go:embed client.cnf
var clientConfig string

const (
	pathToNomad = "/usr/local/percona/pmm/tools/nomad"
	pathToCerts = "/srv/nomad/certs"
	region      = "global"
)

// Nomad is a wrapper around Nomad client.
type Nomad struct {
	prefix string
	domain string
}

// New creates a new Nomad client.
func New(domain string) (*Nomad, error) {
	err := os.MkdirAll(pathToCerts, 0o755)
	if err != nil {
		return nil, err
	}
	return &Nomad{
		prefix: "nomad",
		domain: domain,
	}, nil
}

// GenerateCACert generates a new CA certificate.
func (c *Nomad) GenerateCACert() error {
	command := exec.Command(pathToNomad, "ca", "create", "-days", "10000")
	command.Dir = pathToCerts
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) GenerateServerCert() error {
	command := exec.Command(pathToNomad,
		"cert",
		"create",
		"-server",
		"-days", "10000",
		"-ca", c.pathToCA(),
		"-key", c.pathToCAKey(),
		"-domain", c.domain,
		"region", region,
	)
	command.Dir = pathToCerts
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *Nomad) GenerateClientCert() error {
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
