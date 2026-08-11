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
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/clickhouse"
	"github.com/percona/pmm/managed/services/supervisord"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/env"
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
	clickHouseConfig, err := clickhouse.GetClickHouseConfig(os.Getenv(env.ClickHouseConfig))
	if err != nil {
		errs = append(errs, err)
	}
	for _, warn := range warns {
		logrus.Warnf("Configuration warning: %s", warn)
	}
	for _, err := range errs {
		logrus.Errorf("Configuration error: %s", err)
	}
	if len(errs) != 0 {
		os.Exit(1)
	}

	err = models.ValidateSettings(envSettings)
	if err != nil {
		logrus.Errorf("Configuration error: %s.", err)
		os.Exit(1)
	}

	pmmConfigParams := make(map[string]any)
	pmmConfigParams["DisableInternalDB"], _ = strconv.ParseBool(os.Getenv("PMM_DISABLE_BUILTIN_POSTGRES"))
	pmmConfigParams["DisableInternalClickhouse"], _ = strconv.ParseBool(os.Getenv("PMM_DISABLE_BUILTIN_CLICKHOUSE"))
	pmmConfigParams["AgentConfigFilePath"] = models.AgentConfigFilePath
	pmmConfigParams["ClickHouseConfig"] = clickHouseConfig

	isHAEnabled, _ := strconv.ParseBool(os.Getenv("PMM_HA_ENABLE"))
	if isHAEnabled {
		pmmConfigParams["AgentConfigFilePath"] = "/srv/pmm-agent/config/pmm-agent.yaml"

		err = checkHAEncryptionKey()
		if err != nil {
			logrus.Errorf("Configuration error: %s", err)
			os.Exit(1)
		}
	}

	err = supervisord.SavePMMConfig(pmmConfigParams)
	if err != nil {
		logrus.Errorf("PMM Server configuration error: %s.", err)
		os.Exit(1)
	}
}

// checkHAEncryptionKey refuses to start an HA node that has no encryption key yet.
//
// All nodes of an HA cluster share one PostgreSQL database but keep their own key file, so
// letting a node generate its own key leaves it unable to decrypt rows written by the others.
func checkHAEncryptionKey() error {
	path := encryption.KeyPath()

	_, err := os.Stat(path)
	switch {
	case err == nil:
		return nil
	case os.IsNotExist(err):
		return fmt.Errorf("encryption key %s not found. In HA mode all PMM Server nodes must share "+
			"one encryption key, so it is never generated automatically. Generate it once with "+
			"`pmm-encryption-rotation --generate-key`, place the output at %s on every node, then start them",
			path, path)
	default:
		return fmt.Errorf("cannot read encryption key %s: %w", path, err)
	}
}
