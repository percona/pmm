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

	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-update/yum"
)

func check() {
	v, err := yum.CheckVersions(context.Background(), "pmm-update")
	if err != nil {
		logrus.Tracef("%+v", err)
		logrus.Fatalf("CheckVersions failed: %s", err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(v); err != nil {
		logrus.Fatal(err)
	}
}

func perform() {
}

// Flags have to be global variables for maincover_test.go to work.
//nolint:gochecknoglobals
var (
	checkF   = flag.Bool("check", false, "Check for updates")
	performF = flag.Bool("perform", false, "Perform update")
	debugF   = flag.Bool("debug", false, "Enable debug logging")
	traceF   = flag.Bool("trace", false, "Enable trace logging")
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

	if *checkF {
		check()
	}
	if *performF {
		perform()
	}
}
