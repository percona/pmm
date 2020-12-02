// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona-platform/saas/pkg/starlark"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.starlark.net/resolve"
	"golang.org/x/sys/unix"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-managed/services/checks"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	cpuLimit         = 4 * time.Second
	memoryLimitBytes = 1024 * 1024 * 1024

	// only used for testing.
	starlarkRecursionFlag = "PERCONA_TEST_STARLARK_ALLOW_RECURSION"

	// warning messages.
	cpuUsageWarning    = "Failed to limit CPU usage"
	memoryUsageWarning = "Failed to limit memory usage"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("stdlog: ")

	kingpin.Version(version.FullInfo())
	kingpin.HelpFlag.Short('h')

	kingpin.Parse()

	logger.SetupGlobalLogger()
	if on, _ := strconv.ParseBool(os.Getenv("PMM_DEBUG")); on {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if on, _ := strconv.ParseBool(os.Getenv("PMM_TRACE")); on {
		logrus.SetLevel(logrus.TraceLevel)
	}
	if on, _ := strconv.ParseBool(os.Getenv(starlarkRecursionFlag)); on {
		resolve.AllowRecursion = true
	}

	l := logrus.WithField("component", "pmm-managed-starlark")

	err := unix.Setrlimit(unix.RLIMIT_CPU, &unix.Rlimit{
		Cur: uint64(cpuLimit.Seconds()),
		Max: uint64(cpuLimit.Seconds()),
	})
	if err != nil {
		l.Warnf("%s: %s", cpuUsageWarning, err)
	}
	err = unix.Setrlimit(unix.RLIMIT_DATA, &unix.Rlimit{
		Cur: memoryLimitBytes,
		Max: memoryLimitBytes,
	})
	if err != nil {
		l.Warnf("%s: %s", memoryUsageWarning, err)
	}

	decoder := json.NewDecoder(os.Stdin)
	var data checks.StarlarkScriptData
	err = decoder.Decode(&data)
	if err != nil {
		l.Errorf("Error decoding json data: %s", err)
		os.Exit(1)
	}

	results, err := runChecks(l, &data)
	if err != nil {
		l.Errorf("Error running starlark script: %+v", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	err = encoder.Encode(results)
	if err != nil {
		l.Errorf("Error encoding JSON results: %s", err)
		os.Exit(1)
	}
}

func runChecks(l *logrus.Entry, data *checks.StarlarkScriptData) ([]check.Result, error) {
	funcs, err := checks.GetFuncsForVersion(data.Version)
	if err != nil {
		return nil, errors.Wrap(err, "error getting funcs")
	}

	env, err := starlark.NewEnv(data.Name, data.Script, funcs)
	if err != nil {
		return nil, errors.Wrap(err, "error initializing starlark env")
	}

	input, err := agentpb.UnmarshalActionQueryResult(data.QueryResult)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling query result")
	}

	contextFuncs := checks.GetAdditionalContext()
	results, err := env.Run(data.Name, input, contextFuncs, l.Debugln)
	if err != nil {
		return nil, errors.Wrap(err, "error running starlark env")
	}

	return results, nil
}
