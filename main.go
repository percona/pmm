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

package main

import (
	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-agent/commands"
	"github.com/percona/pmm-agent/config"
)

func main() {
	// empty version breaks much of pmm-managed logic
	if version.Version == "" {
		panic("pmm-agent version is not set during build.")
	}

	// check that command-line flags and environment variables are correct,
	// parse command, but do try not load config file
	cfg := new(config.Config)
	app, _ := config.Application(cfg)
	kingpin.CommandLine = app
	kingpin.HelpFlag = app.HelpFlag
	kingpin.HelpCommand = app.HelpCommand
	kingpin.VersionFlag = app.VersionFlag
	cmd := kingpin.Parse()

	// common logger settings for all commands
	logrus.SetReportCaller(false) // https://github.com/sirupsen/logrus/issues/954
	if cfg.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if cfg.Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true) // https://github.com/sirupsen/logrus/issues/954
	}

	switch cmd {
	case "run":
		commands.Run()
	case "setup":
		commands.Setup()
	default:
		// not reachable due to default kingpin's termination handler; keep it just in case
		kingpin.Fatalf("Unexpected command %q.", cmd)
	}
}
