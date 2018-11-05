// pmm-agent
// Copyright (C) 2018 Percona LLC
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

// Package config provides access to pmm-agent configuration.
package config

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

// Config represents pmm-agent's static configuration.
type Config struct {
	Address string
	Debug   bool

	WithoutNginx bool // FIXME remove this before 2.0.0-proto

	Paths struct {
		MySQLdExporter string
		RDSExporter    string
	}

	UUID string
}

func Application(cfg *Config, version string) *kingpin.Application {
	app := kingpin.New("pmm-agent", "Version "+version+".")
	app.HelpFlag.Short('h')
	app.Version(version)
	app.Flag("address", "PMM Server address.").Envar("PMM_AGENT_ADDRESS").StringVar(&cfg.Address)
	app.Flag("debug", "Enable debug output.").Envar("PMM_AGENT_DEBUG").BoolVar(&cfg.Debug)
	app.Flag("uuid", "UUID of this pmm-agent.").Envar("PMM_AGENT_UUID").StringVar(&cfg.UUID)

	app.Flag("without-nginx", "Connect directly to pmm-managed, not via nginx.").BoolVar(&cfg.WithoutNginx)

	// TODO load configuration from file with kingpin.ExpandArgsFromFile
	// TODO show environment variables in help

	return app
}
