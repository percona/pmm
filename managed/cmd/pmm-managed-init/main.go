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
	"cmp"
	"errors"
	"fmt"
	"io/fs"
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

	if len(os.Args) > 1 && os.Args[1] == vaultEnvCommand {
		err := runVaultEnv()
		if err != nil {
			logrus.Errorf("Configuration error: %s.", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) == 3 && os.Args[1] == vaultPutCommand {
		err := runVaultPut(os.Args[2])
		if err != nil {
			logrus.Errorf("Failed to store the secret in Vault: %s.", err)
			os.Exit(1)
		}
		return
	}
	envSettings, errs, warns := envvars.ParseEnvVars(os.Environ())
	disableInternalClickhouse, _ := strconv.ParseBool(os.Getenv("PMM_DISABLE_BUILTIN_CLICKHOUSE"))
	clickHouseConfig, err := clickhouse.GetClickHouseConfig(os.Getenv(env.ClickHouseConfig))
	if err != nil {
		errs = append(errs, err)
	}
	// An external ClickHouse leaves the built-in server unused, so its config files are irrelevant.
	if err == nil && !disableInternalClickhouse {
		linkErr := clickhouse.LinkClickHouseConfig(clickHouseConfig)
		if linkErr != nil {
			errs = append(errs, linkErr)
		}
	}
	if !disableInternalClickhouse {
		passwordsErr := clickhouse.ApplyPasswords(os.Getenv)
		if passwordsErr != nil {
			errs = append(errs, passwordsErr)
		}
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
	pmmConfigParams["DisableInternalClickhouse"] = disableInternalClickhouse
	pmmConfigParams["AgentConfigFilePath"] = models.AgentConfigFilePath

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

	err = saveVMCredentials()
	if err != nil {
		logrus.Errorf("PMM Server configuration error: %s.", err)
		os.Exit(1)
	}
}

// saveVMCredentials writes the credentials that PMM_VM_URL carries to the files that vmalert and
// vmproxy read them from. It runs before supervisord starts them, since vmalert does not start
// without the files that its configuration from the previous start names.
func saveVMCredentials() error {
	// pmm-managed ignores an empty PMM_VM_URL too.
	vmURL, err := models.ParseVictoriaMetricsURL(cmp.Or(os.Getenv("PMM_VM_URL"), models.VMBaseURL))
	if err != nil {
		return err
	}
	return supervisord.SaveVMCredentials(vmURL)
}

// checkHAEncryptionKey refuses to start an HA node that has no encryption key yet.
//
// All nodes of an HA cluster share one PostgreSQL database but keep their own key file, so
// letting a node generate its own key leaves it unable to decrypt rows written by the others.
func checkHAEncryptionKey() error {
	// Given as a reference to the key in Vault, as a rule, which every node reads the same.
	if os.Getenv(encryption.KeyEnvVar) != "" {
		return nil
	}

	path := encryption.KeyPath()

	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("encryption key %s not found. In HA mode all PMM Server nodes must share "+
			"one encryption key, so it is never generated automatically. Generate it once with "+
			"`pmm-encryption-rotation --generate-key`, place the output at %s on every node, then start them",
			path, path)
	case err != nil:
		return fmt.Errorf("cannot read encryption key %s: %w", path, err)
	case !info.Mode().IsRegular() || info.Size() == 0:
		return fmt.Errorf("encryption key %s is empty or not a regular file. Place the key shared by "+
			"all PMM Server nodes there, generated once with `pmm-encryption-rotation --generate-key`", path)
	default:
		return nil
	}
}
