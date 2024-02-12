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
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/update/pkg/ansible"
	"github.com/percona/pmm/update/pkg/yum"
	"github.com/percona/pmm/version"
)

const (
	pmmManagedPackageName = "pmm-managed"
)

func installed(ctx context.Context) {
	v, err := yum.Installed(ctx, pmmManagedPackageName)
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("Installed failed: %s", err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(v); err != nil {
		logrus.Fatal(err)
	}
}

func check(ctx context.Context) {
	pmmManagedPackage, err := yum.Check(ctx, pmmManagedPackageName)
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("Check failed: %s", err)
	}

	if err = json.NewEncoder(os.Stdout).Encode(pmmManagedPackage); err != nil {
		logrus.Fatal(err)
	}
}

func performStage1SelfUpdate(ctx context.Context) {
	const name = "pmm-update"
	v, err := yum.Installed(ctx, name)
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("Installed failed before update: %s", err)
	}
	before := v.Installed

	if err = yum.Update(ctx, name); err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("Update failed: %s", err)
	}

	v, err = yum.Installed(ctx, name)
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("Installed failed after update: %s", err)
	}
	after := v.Installed

	logrus.Infof("%s:\nbefore update = %+v\n after update = %+v", name, before, after)
	if before.FullVersion != after.FullVersion {
		// exit with non-zero code to let supervisord restart `pmm-update -perform` from the start
		logrus.Info("Version changed, exiting.")
		os.Exit(42)
	}
	logrus.Info("Version did not change.")
}

func performStage2Ansible(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	err := ansible.RunPlaybook(ctx, playbook, opts)
	if err != nil {
		logrus.Fatalf("RunPlaybook failed: %s", err)
	}
}

func runAnsible(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	performStage2Ansible(ctx, playbook, opts)
}

func perform(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	performStage1SelfUpdate(ctx)
	performStage2Ansible(ctx, playbook, opts)

	// pmm-managed will still wait for dashboard-upgrade to finish;
	// that string is expected by various automated tests.
	logrus.Info("Waiting for Grafana dashboards update to finish...")
}

// Flags have to be global variables for maincover_test.go to work.
//
//nolint:gochecknoglobals
var (
	installedF   = flag.Bool("installed", false, "Return installed version")
	checkF       = flag.Bool("check", false, "Check for updates")
	performF     = flag.Bool("perform", false, "Perform update")
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
	if *installedF {
		modes++
	}
	if *checkF {
		modes++
	}
	if *performF {
		modes++
	}
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
	case *installedF:
		installed(ctx)
	case *checkF:
		check(ctx)
	case *runPlaybookF:
		if *playbookF == "" {
			logrus.Fatalf("-playbook flag must be set.")
		}

		runAnsible(ctx, *playbookF, &ansible.RunPlaybookOpts{
			Debug: *debugF,
			Trace: *traceF,
		})
	case *performF:
		if *playbookF == "" {
			logrus.Fatalf("-playbook flag must be set.")
		}
		perform(ctx, *playbookF, &ansible.RunPlaybookOpts{
			Debug: *debugF,
			Trace: *traceF,
		})
	}
}
