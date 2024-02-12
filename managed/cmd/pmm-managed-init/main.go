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
// Package main provides the entry point for the pmm-managed-init application.
package main

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/supervisord"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/utils/logger"
)

func main() {
	logger.SetupGlobalLogger()
	if on, _ := strconv.ParseBool(os.Getenv("PMM_DEBUG")); on {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if on, _ := strconv.ParseBool(os.Getenv("PMM_TRACE")); on {
		logrus.SetLevel(logrus.TraceLevel)
	}
	envSettings, errs, warns := envvars.ParseEnvVars(os.Environ())
	for _, warn := range warns {
		logrus.Warnf("Configuration warning: %s.", warn)
	}
	for _, err := range errs {
		logrus.Errorf("Configuration error: %s.", err)
	}
	if len(errs) != 0 {
		os.Exit(1)
	}

	err := models.ValidateSettings(envSettings)
	if err != nil {
		logrus.Errorf("Configuration error: %s.", err)
		os.Exit(1)
	}

	pmmConfigParams := make(map[string]any)
	pmmConfigParams["DisableInternalDB"], _ = strconv.ParseBool(os.Getenv("PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES"))
	pmmConfigParams["DisableInternalClickhouse"], _ = strconv.ParseBool(os.Getenv("PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE"))
	if err := supervisord.SavePMMConfig(pmmConfigParams); err != nil {
		logrus.Errorf("PMM Server configuration error: %s.", err)
		os.Exit(1)
	}
}
