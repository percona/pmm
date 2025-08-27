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
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/supervisord"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const (
	// Version file paths as defined in the ansible initialization role
	currentVersionFile = "/srv/grafana/PERCONA_DASHBOARDS_VERSION"
	imageVersionFile   = "/usr/share/percona-dashboards/VERSION"
	// Environment variable to allow downgrades
	envAllowDowngrade = "PMM_ALLOW_DOWNGRADE"
)

// readAndParseVersion reads a version file and parses it, returning the parsed version.
func readAndParseVersion(filePath string) (version.Parsed, error) {
	versionBytes, err := os.ReadFile(filePath)
	if err != nil {
		return version.Parsed{}, err
	}

	versionStr := strings.TrimSpace(string(versionBytes))
	parsedVersion, err := version.Parse(versionStr)
	if err != nil {
		return version.Parsed{}, err
	}

	return *parsedVersion, nil
}

// checkForDowngrade checks if the current version in /srv is newer than the image version
// and fails with an error if a downgrade is attempted.
// Can be overridden with PMM_ALLOW_DOWNGRADE=true environment variable.
func checkForDowngrade() error {
	// Check if downgrade is explicitly allowed
	if allowDowngrade, _ := strconv.ParseBool(os.Getenv(envAllowDowngrade)); allowDowngrade {
		logrus.Warn("PMM_ALLOW_DOWNGRADE is set - downgrade protection is disabled. This may cause data corruption or system instability.")
		return nil
	}

	// Read and parse current version
	currentVersion, err := readAndParseVersion(currentVersionFile)
	if os.IsNotExist(err) {
		// If current version file doesn't exist, this is a fresh install - allow it
		logrus.Debug("No existing version found, this appears to be a fresh installation")
		return nil
	}
	if err != nil {
		logrus.Warnf("Failed to parse current version: %v", err)
		return nil // Allow operation to proceed if we can't parse current version
	}

	// Read and parse image version
	imageVersion, err := readAndParseVersion(imageVersionFile)
	if err != nil {
		return err
	}

	logrus.Debugf("Current version: %s, Image version: %s", currentVersion.String(), imageVersion.String())

	// Check for downgrade: if current version is newer than image version, fail
	if imageVersion.Less(&currentVersion) {
		return errors.Errorf("Downgrade detected: current version %s is newer than image version %s. "+
			"Downgrades are not supported and could cause data corruption or system instability.",
			currentVersion.String(), imageVersion.String())
	}

	logrus.Info("Version check passed")
	return nil
}

func main() {
	logger.SetupGlobalLogger()
	if on, _ := strconv.ParseBool(os.Getenv("PMM_DEBUG")); on {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if on, _ := strconv.ParseBool(os.Getenv("PMM_TRACE")); on {
		logrus.SetLevel(logrus.TraceLevel)
	}

	// Check for downgrade before proceeding with initialization
	if err := checkForDowngrade(); err != nil {
		logrus.Errorf("Downgrade check failed: %s", err)
		os.Exit(1)
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
	pmmConfigParams["DisableInternalDB"], _ = strconv.ParseBool(os.Getenv("PMM_DISABLE_BUILTIN_POSTGRES"))
	pmmConfigParams["DisableInternalClickhouse"], _ = strconv.ParseBool(os.Getenv("PMM_DISABLE_BUILTIN_CLICKHOUSE"))
	if err := supervisord.SavePMMConfig(pmmConfigParams); err != nil {
		logrus.Errorf("PMM Server configuration error: %s.", err)
		os.Exit(1)
	}
}
