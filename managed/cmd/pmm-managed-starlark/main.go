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
// Package main.
package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona-platform/saas/pkg/starlark"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.starlark.net/resolve"
	"golang.org/x/sys/unix"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/services/checks"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const (
	cpuLimit         = 4 * time.Second
	memoryLimitBytes = 1024 * 1024 * 1024

	// Only used for testing.
	starlarkRecursionFlag = "PERCONA_TEST_STARLARK_ALLOW_RECURSION"

	// Warning messages.
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

	res := make([]any, len(data.QueriesResults))
	for i, queryResult := range data.QueriesResults {
		switch qr := queryResult.(type) {
		case map[string]any: // used for PG multidb results where key is database name and value is rows
			dbRes := make(map[string]any, len(qr))
			for dbName, dbQr := range qr {
				s, ok := dbQr.(string)
				if !ok {
					return nil, errors.Errorf("unexpected query result type: %T", dbQr)
				}
				if dbRes[dbName], err = unmarshalQueryResult(s); err != nil {
					return nil, err
				}
			}
			res[i] = dbRes
		case string: // used for all other databases
			if res[i], err = unmarshalQueryResult(qr); err != nil {
				return nil, err
			}
		default:
			return nil, errors.Errorf("unknown query result type %T", qr)
		}
	}

	var results []check.Result
	contextFuncs := checks.GetAdditionalContext()
	switch data.Version {
	case 1:
		results, err = env.Run(data.Name, res[0], contextFuncs, l.Debugln)
	case 2:
		results, err = env.Run(data.Name, res, contextFuncs, l.Debugln)
	}
	if err != nil {
		return nil, errors.Wrap(err, "error running starlark env")
	}

	return results, nil
}

func unmarshalQueryResult(qr string) ([]map[string]any, error) {
	b, err := base64.StdEncoding.DecodeString(qr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 encoded query result")
	}

	res, err := agentpb.UnmarshalActionQueryResult(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal query result")
	}

	return res, nil
}
