// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package main.
package main

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm/agent/commands"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

func main() {
	// empty version breaks much of pmm-managed logic
	if version.Version == "" {
		panic("pmm-agent version is not set during build.")
	}

	logger.SetupGlobalLogger()

	// check that command-line flags and environment variables are correct,
	// parse command, but do try not load config file
	var cfg config.Config
	app, _ := config.Application(&cfg)
	kingpin.CommandLine = app
	kingpin.HelpFlag = app.HelpFlag
	kingpin.HelpCommand = app.HelpCommand
	cmd := kingpin.Parse()

	switch cmd {
	case "run":
		// delay logger configuration until we read configuration file
		commands.Run()
	case "setup":
		config.ConfigureLogger(&cfg)
		commands.Setup()
	default:
		// not reachable due to default kingpin's termination handler; keep it just in case
		kingpin.Fatalf("Unexpected command %q.", cmd)
	}
}
