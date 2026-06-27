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

// Package envvars contains environment variables parser.
package envvars

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	pkgenv "github.com/percona/pmm/managed/utils/env"
)

const (
	defaultPlatformAddress    = "https://check.percona.com"
	defaultPlatformAPITimeout = 30 * time.Second
	// EnvVMAgentPrefix is the prefix for environment variables related to the VMAgent.
	EnvVMAgentPrefix = "VMAGENT_"
	// EnvVMAuthPrefix is the prefix for environment variables related to VMAuth.
	EnvVMAuthPrefix = "VMAUTH_"
	// EnvVMSelectPrefix is the prefix for environment variables related to VMSelect.
	EnvVMSelectPrefix = "VMSELECT_"
	// EnvVMInsertPrefix is the prefix for environment variables related to VMInsert.
	EnvVMInsertPrefix = "VMINSERT_"
	// EnvVMStoragePrefix is the prefix for environment variables related to VMStorage.
	EnvVMStoragePrefix = "VMSTORAGE_"
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
//   - PATH, HOSTNAME, TERM, HOME are default environment variables that will be ignored;
//   - PMM_ENABLE_UPDATES is a boolean flag to enable or disable pmm-server update;
//   - PMM_ENABLE_TELEMETRY is a boolean flag to enable or disable pmm telemetry (and disable Advisors if telemetry is disabled);
//   - PMM_ENABLE_ALERTING disables Percona Alerting;
//   - PMM_METRICS_RESOLUTION, PMM_METRICS_RESOLUTION_MR, PMM_METRICS_RESOLUTION_HR, PMM_METRICS_RESOLUTION_LR are durations of metrics resolution;
//   - PMM_DATA_RETENTION is the duration of how long keep time-series data in ClickHouse;
//   - PMM_ENABLE_AZURE_DISCOVER enables Azure Discover;
//   - PMM_ENABLE_ACCESS_CONTROL enables Access control;
//   - the environment variables prefixed with GF_ passed as related to Grafana.
//   - the environment variables relating to proxies
//   - the environment variable set by podman
func ParseEnvVars(envs []string) (*models.ChangeSettingsParams, []error, []string) { //nolint:gocognit,cyclop,maintidx
	envSettings := &models.ChangeSettingsParams{}
	var errs []error
	var warns []string

	for _, env := range envs {
		p := strings.SplitN(env, "=", 2) //nolint:mnd

		if len(p) != 2 { //nolint:mnd
			errs = append(errs, fmt.Errorf("failed to parse environment variable %q", env))
			continue
		}

		k, v := strings.ToUpper(p[0]), strings.ToLower(p[1])
		logrus.Tracef("ParseEnvVars: %#q: k=%#q v=%#q", env, k, v)

		var err error
		switch k {
		case "_", "HOME", "HOSTNAME", "LANG", "PATH", "PWD", "SHLVL", "TERM", "LC_ALL", "SHELL", "LOGNAME", "USER", "PS1":
			// skip default environment variables
			continue
		case "NO_PROXY", "HTTP_PROXY", "HTTPS_PROXY":
			continue
		case "CONTAINER":
			continue
		case "NSS_WRAPPER_GROUP", "NSS_WRAPPER_PASSWD", "LD_PRELOAD":
			// skip nss_wrapper environment variables
			continue
		case "AWS_ACCESS_KEY", "AWS_SECRET_KEY":
			continue

		case "PMM_DEBUG", "PMM_TRACE":
			// skip cross-component environment variables that are already handled by kingpin
			continue
		case "PMM_CLICKHOUSE_DATABASE", "PMM_CLICKHOUSE_ADDR",
			"PMM_CLICKHOUSE_USER", "PMM_CLICKHOUSE_PASSWORD",
			"PMM_CLICKHOUSE_HOST", "PMM_CLICKHOUSE_PORT",
			"PMM_CLICKHOUSE_IS_CLUSTER", "PMM_CLICKHOUSE_CLUSTER_NAME",
			"PMM_CLICKHOUSE_NODES", "PMM_DISABLE_BUILTIN_CLICKHOUSE",
			pkgenv.ClickHouseConfig:
			continue
		case "PMM_POSTGRES_ADDR",
			"PMM_POSTGRES_DBNAME",
			"PMM_POSTGRES_USERNAME",
			"PMM_POSTGRES_DBPASSWORD",
			"PMM_POSTGRES_SSL_MODE",
			"PMM_POSTGRES_SSL_CA_PATH",
			"PMM_POSTGRES_SSL_KEY_PATH",
			"PMM_POSTGRES_SSL_CERT_PATH",
			"PMM_DISABLE_BUILTIN_POSTGRES":
			// skip env variables for external postgres
			continue
		case "PMM_WATCHTOWER_TOKEN", "PMM_WATCHTOWER_HOST":
			// skip watchtower environement variables
			continue
		case "PERCONA_TELEMETRY_DISABLE":
			// skip the Pillars telemetry environment variable
			continue
		case "PMM_ENABLE_UPDATES":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableUpdates = &b
		case "PMM_UPDATE_SNOOZE_DURATION":
			envSettings.UpdateSnoozeDuration, err = parseStringDuration(v)
			if err != nil {
				errs = append(errs, formatEnvVariableError(err, env, v))
				continue
			}
		case "PMM_ENABLE_TELEMETRY":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableTelemetry = &b
		case pkgenv.EnableInternalPgQAN:
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableInternalPgQAN = &b
		case "PMM_METRICS_RESOLUTION", "PMM_METRICS_RESOLUTION_HR":
			envSettings.MetricsResolutions.HR, err = parseStringDuration(v)
			if err != nil {
				errs = append(errs, formatEnvVariableError(err, env, v))
				continue
			}
		case "PMM_METRICS_RESOLUTION_MR":
			envSettings.MetricsResolutions.MR, err = parseStringDuration(v)
			if err != nil {
				errs = append(errs, formatEnvVariableError(err, env, v))
				continue
			}
		case "PMM_METRICS_RESOLUTION_LR":
			envSettings.MetricsResolutions.LR, err = parseStringDuration(v)
			if err != nil {
				errs = append(errs, formatEnvVariableError(err, env, v))
				continue
			}
		case "PMM_DATA_RETENTION":
			envSettings.DataRetention, err = parseStringDuration(v)
			if err != nil {
				errs = append(errs, formatEnvVariableError(err, env, v))
				continue
			}
		case "PMM_ENABLE_VM_CACHE":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableVMCache = &b
		case "PMM_ENABLE_ALERTING":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableAlerting = &b

		case "PMM_ENABLE_AZURE_DISCOVER":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableAzurediscover = &b

		case "PMM_ENABLE_BACKUP_MANAGEMENT":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableBackupManagement = &b

		case "PMM_ENABLE_NOMAD":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableNomad = &b

		case "PMM_PUBLIC_ADDRESS":
			envSettings.PMMPublicAddress = new(v)

		case "PMM_VM_URL":
			_, err = url.Parse(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
			}

		case "PMM_INSTALL_METHOD", "PMM_DISTRIBUTION_METHOD":
			continue

		// skip various HA-related variables
		case "PMM_ENCRYPTION_KEY_PATH", "PMM_ADMIN_PASSWORD":
			continue

		case pkgenv.EnableAccessControl:
			b, err := strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("invalid value %q for environment variable %q", v, k)
				errs = append(errs, err)
				continue
			}

			envSettings.EnableAccessControl = &b

		case pkgenv.PlatformAPITimeout:
			// This variable is not part of the settings and is parsed separately.
			continue

		default:
			// handle prefixes

			// skip Grafana's environment variables
			if strings.HasPrefix(k, "GF_") {
				continue
			}

			// skip Victoria Metrics' environment variables
			if strings.HasPrefix(k, "VM_") {
				continue
			}

			// skip VM Agents environment variables
			if strings.HasPrefix(k, EnvVMAgentPrefix) {
				continue
			}

			// skip VMAuth environment variables
			if strings.HasPrefix(k, EnvVMAuthPrefix) {
				continue
			}

			// skip VMSelect environment variables
			if strings.HasPrefix(k, EnvVMSelectPrefix) {
				continue
			}

			// skip VMInsert environment variables
			if strings.HasPrefix(k, EnvVMInsertPrefix) {
				continue
			}

			// skip VMStorage environment variables
			if strings.HasPrefix(k, EnvVMStoragePrefix) {
				continue
			}

			// skip supervisord environment variables
			if strings.HasPrefix(k, "SUPERVISOR_") {
				continue
			}

			// skip kubernetes environment variables
			if strings.HasPrefix(k, "KUBERNETES_") || strings.HasPrefix(k, "PMM_OPERATORS_") {
				continue
			}

			// skip kubernetes monitoring environment variables
			if strings.HasPrefix(k, "MONITORING_") {
				continue
			}

			// skip PMM development environment variables
			if strings.HasPrefix(k, "PMM_DEV_") {
				continue
			}

			// skip PMM HA environment variables
			if strings.HasPrefix(k, "PMM_HA_") {
				continue
			}

			// skip PMM test environment variables
			if strings.HasPrefix(k, "PMM_TEST_") {
				warns = append(warns, fmt.Sprintf("environment variable %s may be removed or replaced in the future", env))
				continue
			}

			if strings.HasPrefix(k, "PERCONA_") {
				warns = append(warns, "PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation")
				continue
			}

			warns = append(warns, "unknown environment variable "+env)
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

func parsePlatformAPITimeout(d string) (time.Duration, string) {
	if d == "" {
		msg := fmt.Sprintf("Setting the default timeout for Platform API to %s.", defaultPlatformAPITimeout.String())
		return defaultPlatformAPITimeout, msg
	}
	duration, err := parseStringDuration(d)
	if err != nil {
		msg := fmt.Sprintf("Set the default Platform API to %s: failed to parse timeout %s: %s.", defaultPlatformAPITimeout.String(), d, err)
		return defaultPlatformAPITimeout, msg
	}
	msg := fmt.Sprintf("Set the timeout for Platform API to %s.", duration.String())
	return duration, msg
}

// GetPlatformAPITimeout returns timeout duration for requests to Platform.
func GetPlatformAPITimeout(l *logrus.Entry) time.Duration {
	d := os.Getenv(pkgenv.PlatformAPITimeout)
	duration, msg := parsePlatformAPITimeout(d)
	l.Info(msg)
	return duration
}

// GetPlatformAddress returns Percona Platform address env variable value if it's present and valid.
// Otherwise returns default Percona Platform address.
func GetPlatformAddress() (string, error) {
	address := os.Getenv(pkgenv.PlatformAddress)
	if address == "" {
		logrus.Infof("Using default Percona Platform address: %s.", defaultPlatformAddress)
		return defaultPlatformAddress, nil
	}

	_, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("invalid Percona Platform address: %w", err)
	}

	logrus.Infof("Using Percona Platform address: %s.", address)
	return address, nil
}

// GetPlatformInsecure returns true if invalid/self-signed TLS certificates allowed. Default is false.
func GetPlatformInsecure() bool {
	insecure, _ := strconv.ParseBool(os.Getenv(pkgenv.PlatformInsecure))

	return insecure
}

// GetInterfaceToBind retrieves the network interface to bind based on environment variables.
func GetInterfaceToBind() string {
	return GetEnv(pkgenv.InterfaceToBind, "127.0.0.1")
}

// GetEnv returns env with fallback option.
func GetEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if ok && value != "" {
		return value
	}
	return fallback
}

func formatEnvVariableError(err error, env, value string) error {
	switch err.(type) { //nolint:errorlint
	case InvalidDurationError:
		return fmt.Errorf("environment variable %q has invalid duration %s", env, value)
	default:
		return fmt.Errorf("unknown error: %w", err)
	}
}
