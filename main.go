// pmm-agent
// Copyright 2019 Percona LLC
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
