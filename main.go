// pmm-update
// Copyright (C) 2019 Percona LLC
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

package main // import "github.com/percona/pmm-update"

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm-update/pkg/ansible"
	"github.com/percona/pmm-update/pkg/yum"
)

func check(ctx context.Context) {
	v, err := yum.CheckVersions(ctx, "pmm-update")
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("CheckVersions failed: %s", err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(v); err != nil {
		logrus.Fatal(err)
	}
}

func performStage1SelfUpdate(ctx context.Context) {
	const name = "pmm-update"
	v, err := yum.CheckVersions(ctx, name)
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("CheckVersions failed before update: %s", err)
	}
	before := v.InstalledRPMVersion

	if err = yum.UpdatePackage(ctx, name); err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("UpdatePackage failed: %s", err)
	}

	v, err = yum.CheckVersions(ctx, name)
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("CheckVersions failed after update: %s", err)
	}
	after := v.InstalledRPMVersion

	if before != after {
		// exit with non-zero code to let supervisord restart `pmm-update -perform` from the start
		logrus.Infof("%s changed from to %q to %q. Exiting.", name, before, after)
		os.Exit(1)
	}
	logrus.Infof("%s version %q not changed.", name, before)
}

func performStage2Ansible(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	err := ansible.RunPlaybook(ctx, playbook, opts)
	if err != nil {
		logrus.Fatalf("RunPlaybook failed: %s", err)
	}
}

func perform(ctx context.Context, playbook string, opts *ansible.RunPlaybookOpts) {
	performStage1SelfUpdate(ctx)
	performStage2Ansible(ctx, playbook, opts)
}

// Flags have to be global variables for maincover_test.go to work.
//nolint:gochecknoglobals
var (
	checkF    = flag.Bool("check", false, "Check for updates")
	performF  = flag.Bool("perform", false, "Perform update")
	playbookF = flag.String("playbook", "", "Ansible playbook for -perform")
	debugF    = flag.Bool("debug", false, "Enable debug logging")
	traceF    = flag.Bool("trace", false, "Enable trace logging")
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

	if *checkF == *performF {
		logrus.Fatalf("Please select a mode with -check or -perform flag.")
	}

	// handle termination signals
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	if *checkF {
		check(ctx)
	}
	if *performF {
		if *playbookF == "" {
			logrus.Fatalf("-playbook flag must be set.")
		}
		opts := &ansible.RunPlaybookOpts{
			Debug: *debugF,
			Trace: *traceF,
		}
		perform(ctx, *playbookF, opts)
	}
}
