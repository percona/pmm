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
	"fmt"
	"path/filepath"
	"runtime"

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

	// we don't have configuration options for formatter, so set it once there
	logrus.SetFormatter(&logrus.TextFormatter{
		// Enable multiline-friendly formatter in both development (with terminal) and production (without terminal):
		// https://github.com/sirupsen/logrus/blob/839c75faf7f98a33d445d181f3018b5c3409a45e/text_formatter.go#L176-L178
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.000-07:00",

		CallerPrettyfier: func(f *runtime.Frame) (function string, file string) {
			_, function = filepath.Split(f.Function)

			// keep a single directory name as a compromise between brevity and unambiguity
			var dir string
			dir, file = filepath.Split(f.File)
			dir = filepath.Base(dir)
			file = fmt.Sprintf("%s/%s:%d", dir, file, f.Line)

			return
		},
	})

	// check that command-line flags and environment variables are correct,
	// parse command, but do try not load config file
	cfg := new(config.Config)
	app, _ := config.Application(cfg)
	kingpin.CommandLine = app
	kingpin.HelpFlag = app.HelpFlag
	kingpin.HelpCommand = app.HelpCommand
	kingpin.VersionFlag = app.VersionFlag
	cmd := kingpin.Parse()

	switch cmd {
	case "run":
		// delay logger configuration until we read configuration file
		commands.Run()
	case "setup":
		config.ConfigureLogger(cfg)
		commands.Setup()
	default:
		// not reachable due to default kingpin's termination handler; keep it just in case
		kingpin.Fatalf("Unexpected command %q.", cmd)
	}
}
