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
// Package main provides the entry point for the update application.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/update/pkg/ansible"
	"github.com/percona/pmm/version"
)

func performStage2Ansible(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	err := ansible.RunPlaybook(ctx, playbook, opts)
	if err != nil {
		logrus.Fatalf("RunPlaybook failed: %s", err)
	}
}

func runAnsible(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	performStage2Ansible(ctx, playbook, opts)
}

// Flags have to be global variables for maincover_test.go to work.
//
//nolint:gochecknoglobals
var (
	runPlaybookF = flag.Bool("run-playbook", false, "Run playbook without self-update")
	playbookF    = flag.String("playbook", "", "Ansible playbook for -perform")
	debugF       = flag.Bool("debug", false, "Enable debug logging")
	traceF       = flag.Bool("trace", false, "Enable trace logging")
)

func main() {
	log.SetFlags(0)
	log.Print(version.FullInfo())
	log.SetPrefix("stdlog: ")
	flag.Parse()

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *traceF {
		*debugF = *traceF
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true) // https://github.com/sirupsen/logrus/issues/954
	}

	var modes int
	if *runPlaybookF {
		modes++
	}
	if modes != 1 {
		logrus.Fatalf("Please select a mode: -current, -check, -run-playbook or -perform.")
	}

	// handle termination signals
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		cancel()
	}()

	switch {
	case *runPlaybookF:
		if *playbookF == "" {
			logrus.Fatalf("-playbook flag must be set.")
		}

		runAnsible(ctx, *playbookF, &ansible.RunPlaybookOpts{
			Debug: *debugF,
			Trace: *traceF,
		})
	}
}
