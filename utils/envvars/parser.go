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

// Package envvars contains environment variables parser.
package envvars

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-managed/models"
)

const (
	defaultSaaSHost = "check.percona.com:443"
	envSaaSHost     = "PERCONA_TEST_SAAS_HOST"
)

// InvalidDurationError invalid duration error.
type InvalidDurationError string

func (e InvalidDurationError) Error() string { return string(e) }

// ParseEnvVars parses given environment variables.
//
// Returns valid setting and two lists with errors and warnings.
// This function is mainly used in pmm-managed-init to early validate passed
// environment variables, and provide user warnings about unknown variables.
// In case of error, the docker run terminates.
// Short description of environment variables:
//  - PATH, HOSTNAME, TERM, HOME are default environment variables that will be ignored;
//  - DISABLE_UPDATES is a boolean flag to enable or disable pmm-server update;
//  - DISABLE_TELEMETRY is a boolean flag to enable or disable pmm telemetry (and disable STT if telemetry is disabled);
//  - METRICS_RESOLUTION, METRICS_RESOLUTION, METRICS_RESOLUTION_HR, METRICS_RESOLUTION_LR are durations of metrics resolution;
//  - DATA_RETENTION is the duration of how long keep time-series data in ClickHouse;
//  - ENABLE_ALERTING enables Integrated Alerting;
//  - ENABLE_AZUREDISCOVER enables Azure Discover;
//  - the environment variables prefixed with GF_ passed as related to Grafana.
func ParseEnvVars(envs []string) (envSettings *models.ChangeSettingsParams, errs []error, warns []string) {
	envSettings = new(models.ChangeSettingsParams)

	for _, env := range envs {
		p := strings.SplitN(env, "=", 2)

		if len(p) != 2 {
			errs = append(errs, fmt.Errorf("failed to parse environment variable %q", env))
			continue
		}

		k, v := strings.ToUpper(p[0]), strings.ToLower(p[1])
		logrus.Tracef("ParseEnvVars: %#q: k=%#q v=%#q", env, k, v)

		var err error
		switch k {
		case "_", "HOME", "HOSTNAME", "LANG", "PATH", "PWD", "SHLVL", "TERM":
			// skip default environment variables
			continue
		case "PMM_DEBUG", "PMM_TRACE":
			// skip cross-component environment variables that are already handled by kingpin
			continue
		case "DISABLE_UPDATES":
			envSettings.DisableUpdates, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
			}
		case "DISABLE_TELEMETRY":
			envSettings.DisableTelemetry, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
			} else if envSettings.DisableTelemetry {
				// special case
				envSettings.DisableSTT = true
			}
		case "METRICS_RESOLUTION", "METRICS_RESOLUTION_HR":
			if envSettings.MetricsResolutions.HR, err = parseStringDuration(v); err != nil {
				err = formatEnvVariableError(err, env, v)
			}
		case "METRICS_RESOLUTION_MR":
			if envSettings.MetricsResolutions.MR, err = parseStringDuration(v); err != nil {
				err = formatEnvVariableError(err, env, v)
			}
		case "METRICS_RESOLUTION_LR":
			if envSettings.MetricsResolutions.LR, err = parseStringDuration(v); err != nil {
				err = formatEnvVariableError(err, env, v)
			}
		case "DATA_RETENTION":
			if envSettings.DataRetention, err = parseStringDuration(v); err != nil {
				err = formatEnvVariableError(err, env, v)
			}
		case "ENABLE_VM_CACHE":
			envSettings.EnableVMCache, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
			}
			if !envSettings.EnableVMCache {
				// disable cache explicitly
				envSettings.DisableVMCache = true
			}

		case "PERCONA_TEST_IA": // FIXME remove
			warns = append(warns, fmt.Sprintf("Environment variable %q WILL BE REMOVED SOON, please use %q instead.", k, "ENABLE_ALERTING"))
			fallthrough
		case "ENABLE_ALERTING":
			envSettings.EnableAlerting, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
			}
		case "ENABLE_AZUREDISCOVER":
			envSettings.EnableAzurediscover, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
			}
		case "ENABLE_BACKUP_MANAGEMENT":
			envSettings.EnableBackupManagement, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
			}

		case "PERCONA_TEST_AUTH_HOST", "PERCONA_TEST_CHECKS_HOST", "PERCONA_TEST_TELEMETRY_HOST":
			err = fmt.Errorf("environment variable %q is removed and replaced by %q", k, envSaaSHost)

		default:
			// handle prefixes

			// skip Grafana's environment variables
			if strings.HasPrefix(k, "GF_") {
				continue
			}

			// skip supervisord environment variables
			if strings.HasPrefix(k, "SUPERVISOR_") {
				continue
			}

			if !strings.HasPrefix(k, "PERCONA_TEST_") {
				warns = append(warns, fmt.Sprintf("unknown environment variable %q", env))
				continue
			}

			warns = append(warns, fmt.Sprintf("environment variable %q IS NOT SUPPORTED and WILL BE REMOVED IN THE FUTURE", k))

			// TODO rename to ENABLE_DBAAS and move to own `case` before `default:`
			if k == "PERCONA_TEST_DBAAS" {
				envSettings.EnableDBaaS, err = strconv.ParseBool(v)
				if err != nil {
					err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
					errs = append(errs, err)
					continue
				}
				envSettings.DisableDBaaS = !envSettings.EnableDBaaS
			}
		}

		if err != nil {
			errs = append(errs, err)
		}
	}

	return envSettings, errs, warns
}

// parseStringDuration validate duration as string value.
func parseStringDuration(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return d, InvalidDurationError("invalid duration error")
	}

	return d, nil
}

// GetSAASHost returns SaaS host env variable value if it's present and valid.
// Otherwise returns defaultSaaSHost.
func GetSAASHost() (string, error) {
	v := os.Getenv(envSaaSHost)
	host, err := parseSAASHost(v)
	if err != nil {
		return "", err
	}

	logrus.Infof("Using SaaS host %q.", host)
	return host, nil
}

// parseSAASHost parses, validates and returns SAAS host, otherwise returns error.
func parseSAASHost(v string) (string, error) {
	if v == "" {
		logrus.Infof("Using default SaaS host %q.", defaultSaaSHost)
		return defaultSaaSHost, nil
	}
	if strings.HasPrefix(v, ":") {
		return "", fmt.Errorf("environment variable %q has invalid format %q. Expected host[:port]", envSaaSHost, v)
	}

	host, port, err := net.SplitHostPort(v)
	if err != nil && strings.Count(v, ":") >= 1 {
		return "", err
	}
	if host == "" {
		host = v
	}
	if port == "" {
		port = "443"
	}

	v = net.JoinHostPort(host, port)
	return v, nil
}

func formatEnvVariableError(err error, env, value string) error {
	switch e := err.(type) {
	case InvalidDurationError:
		return fmt.Errorf("environment variable %q has invalid duration %s", env, value)
	default:
		return errors.Wrap(e, "unknown error")
	}
}
