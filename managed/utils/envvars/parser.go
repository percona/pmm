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
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	pkgenv "github.com/percona/pmm/managed/utils/env"
)

const (
	defaultPlatformAddress    = "https://check-dev.percona.com"
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
//   - PMM_ENABLE_TELEMETRY is a boolean flag to enable or disable pmm telemetry;
//   - PMM_ENABLE_ALERTING disables Percona Alerting;
//   - PMM_METRICS_RESOLUTION, PMM_METRICS_RESOLUTION_MR, PMM_METRICS_RESOLUTION_HR, PMM_METRICS_RESOLUTION_LR are durations of metrics resolution;
//   - PMM_DATA_RETENTION is the duration of how long keep time-series data in ClickHouse;
//   - PMM_ENABLE_AZURE_DISCOVER enables Azure Discover;
//   - PMM_ENABLE_ACCESS_CONTROL enables Access control;
//   - the environment variables prefixed with GF_ are related to Grafana.
//   - the environment variables prefixed with VMAGENT_ are forwarded to every vmagent PMM Server manages;
//     VMAGENT_remoteWrite_url is validated by checkVMAgentRemoteWriteOverride.
//   - the environment variables related to proxies
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
		logrus.Tracef("ParseEnvVars: k=%#q v=%#q", k, redactSecretEnvVar(k, v))

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
			"PMM_CLICKHOUSE_DATASOURCE_USER", "PMM_CLICKHOUSE_DATASOURCE_PASSWORD",
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
		case "PMM_ENABLE_SEP", "PMM_SEP_POSTGRES_PASSWORD":
			// skip env variables consumed by the entrypoint to expose postgres to SEP
			continue
		case "PERCONA_TELEMETRY_DISABLE":
			// skip the Pillars telemetry environment variable
			continue
		case "PMM_INTERNAL_NODE_NAME_PREFIXES":
			// skip the env variable that is already handled by kingpin
			continue
		case "PMM_ENABLE_UPDATES":
			b, err := strconv.ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value %q for environment variable %q", v, k))
				continue
			}
			envSettings.EnableUpdates = &b
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
			// An empty value is ignored rather than rejected: kingpin falls back to its default
			// for a variable that is set but empty, so this is the address PMM Server uses
			// anyway, and a template that renders an unset value has always started.
			if v == "" {
				warns = append(warns, "PMM_VM_URL is set but empty and is ignored: the built-in VictoriaMetrics address is used")
				continue
			}
			_, err := models.ParseVictoriaMetricsURL(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid value for environment variable %s: %w", k, err))
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

		case pkgenv.PlatformAddress:
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

	overrideErrs, overrideWarns := checkVMAgentRemoteWriteOverride(envs)
	errs = append(errs, overrideErrs...)
	warns = append(warns, overrideWarns...)

	return envSettings, errs, warns
}

// secretEnvVarMarkers name the environment variables whose values must never reach the logs.
var secretEnvVarMarkers = []string{"PASSWORD", "SECRET", "TOKEN", "KEY"}

// redactSecretEnvVar replaces the value of a credential-bearing environment variable.
func redactSecretEnvVar(key, value string) string {
	for _, marker := range secretEnvVarMarkers {
		if strings.Contains(key, marker) {
			return "<redacted>"
		}
	}

	// A URL with userinfo (PMM_VM_URL, VMAGENT_remoteWrite_url): keep scheme and host, drop the
	// credentials.
	return models.RedactURLCredentials(value)
}

// Names of the vmagent remote-write variables PMM Server sets by default. An operator may inject
// them on PMM Server to override the defaults for every vmagent it manages. The password name is
// a variable name, not a secret.
const (
	EnvVMAgentRemoteWriteURL      = "VMAGENT_remoteWrite_url"
	EnvVMAgentRemoteWriteUsername = "VMAGENT_remoteWrite_basicAuth_username"
	EnvVMAgentRemoteWritePassword = "VMAGENT_remoteWrite_basicAuth_password" //nolint:gosec
)

// VMAgentRemoteWriteAuth classifies the remote-write authentication an operator configured
// through VMAGENT_* variables.
type VMAgentRemoteWriteAuth int

const (
	// VMAgentRemoteWriteAuthNone means no authentication variable is set.
	VMAgentRemoteWriteAuthNone VMAgentRemoteWriteAuth = iota
	// VMAgentRemoteWriteAuthPartial means only one half of the basic-auth pair is set.
	VMAgentRemoteWriteAuthPartial
	// VMAgentRemoteWriteAuthComplete means both halves of the basic-auth pair, or another vmagent
	// authentication method, are set.
	VMAgentRemoteWriteAuthComplete
)

// The basic-auth pair by half: vmagent takes each half from a value or from a file, and needs
// one of each.
var (
	remoteWriteUsernameEnvs = []string{EnvVMAgentRemoteWriteUsername, "VMAGENT_remoteWrite_basicAuth_usernameFile"}
	remoteWritePasswordEnvs = []string{EnvVMAgentRemoteWritePassword, "VMAGENT_remoteWrite_basicAuth_passwordFile"}
)

// remoteWriteExclusiveAuthEnvs are the vmagent authentication methods that take the place of a
// basic-auth pair: vmagent refuses to start when one of them is set together with one.
var remoteWriteExclusiveAuthEnvs = []string{
	"VMAGENT_remoteWrite_bearerToken",
	"VMAGENT_remoteWrite_bearerTokenFile",
	"VMAGENT_remoteWrite_oauth2_clientID",
}

// remoteWriteAdditiveAuthEnvs are the vmagent authentication methods that compose with a
// basic-auth pair, such as a tenant header or a client TLS certificate.
var remoteWriteAdditiveAuthEnvs = []string{
	"VMAGENT_remoteWrite_headers",
	"VMAGENT_remoteWrite_tlsCertFile",
}

func envHasAny(env map[string]string, names []string) bool {
	return slices.ContainsFunc(names, func(name string) bool {
		_, ok := env[name]
		return ok
	})
}

// VMAgentRemoteWriteAuthFromEnv classifies the VMAGENT_* variables in env, keyed by name. Names are
// matched case-sensitively, like vmagent matches them.
func VMAgentRemoteWriteAuthFromEnv(env map[string]string) VMAgentRemoteWriteAuth {
	hasUsername := envHasAny(env, remoteWriteUsernameEnvs)
	hasPassword := envHasAny(env, remoteWritePasswordEnvs)

	// The order of the cases is the point. An exclusive method sits above the half-a-pair branch
	// because it legitimately takes the place of a pair, but an additive method sits below it: it
	// composes with a pair instead of replacing one, so it must not report a lone half as complete.
	// VMAgentRemoteWriteReplacesBasicAuth calls that lone half a replacement and withholds PMM's
	// own credential either way, and this is what reports it.
	switch {
	case hasUsername && hasPassword, envHasAny(env, remoteWriteExclusiveAuthEnvs):
		return VMAgentRemoteWriteAuthComplete
	case hasUsername || hasPassword:
		return VMAgentRemoteWriteAuthPartial
	case envHasAny(env, remoteWriteAdditiveAuthEnvs):
		return VMAgentRemoteWriteAuthComplete
	}

	return VMAgentRemoteWriteAuthNone
}

// VMAgentRemoteWriteReplacesBasicAuth reports whether the VMAGENT_* variables in env carry a
// credential that takes the place of a basic-auth pair: any half of a pair of their own, or a
// method vmagent cannot combine with one. Additive methods do not; they leave the pair in place.
func VMAgentRemoteWriteReplacesBasicAuth(env map[string]string) bool {
	return envHasAny(env, remoteWriteUsernameEnvs) || envHasAny(env, remoteWritePasswordEnvs) || envHasAny(env, remoteWriteExclusiveAuthEnvs)
}

// checkVMAgentRemoteWriteOverride validates the operator's VMAGENT_* variables. An empty variable
// is ignored with a warning, because vmagent cannot use an empty value: an empty URL or log level
// stops it and an empty credential disables authentication; buildVMAgentProcess skips it the same
// way. Half a basic-auth pair is always reported. An injected VMAGENT_remoteWrite_url must parse,
// because vmagent refuses to start on a URL it cannot parse; it is reported when no credential
// accompanies it, because PMM's own remote-write credential is not sent to an endpoint PMM did
// not choose. Beyond that the URL is not interpreted: vmagent accepts a comma-separated
// list, and the client renders placeholders such as {{.server_url}} before vmagent starts. The
// vmagent flag names are camelCase and matched case-sensitively both by vmagent and when the
// client config is built, so an upper-cased variant is inert and must not trigger these checks.
func checkVMAgentRemoteWriteOverride(envs []string) ([]error, []string) {
	var warns []string
	vmagentEnv := make(map[string]string)
	for _, env := range envs {
		name, value, ok := strings.Cut(env, "=")
		if !ok || !strings.HasPrefix(name, EnvVMAgentPrefix) {
			continue
		}
		if value == "" {
			warns = append(warns, name+" is set but empty and is ignored: vmagent cannot use an empty value; unset it or give it a value")
			continue
		}
		vmagentEnv[name] = value
	}

	auth := VMAgentRemoteWriteAuthFromEnv(vmagentEnv)
	if auth == VMAgentRemoteWriteAuthPartial {
		warns = append(warns, "only one half of the VMAGENT_remoteWrite_basicAuth_* pair is set "+
			"(a username or usernameFile without a password or passwordFile, or the reverse); set both: "+
			"PMM does not complete the pair with its own default credential, so the lone half is sent alone and fails authentication")
	}

	writeURL, hasURL := vmagentEnv[EnvVMAgentRemoteWriteURL]
	if !hasURL {
		return nil, warns
	}
	parsedURL, err := url.Parse(writeURL)
	if err != nil {
		// The URL may carry a password, so the error must not echo it.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return []error{fmt.Errorf("VMAGENT_remoteWrite_url is not a valid URL, and every vmagent PMM Server manages would refuse to start: %w", err)}, warns
	}
	if auth != VMAgentRemoteWriteAuthNone || parsedURL.User != nil {
		return nil, warns
	}

	return nil, append(warns,
		"VMAGENT_remoteWrite_url redirects the metric writes of every vmagent PMM Server manages to a custom endpoint, "+
			"and PMM's own remote-write credentials are not sent there. Set "+
			"VMAGENT_remoteWrite_basicAuth_username and VMAGENT_remoteWrite_basicAuth_password "+
			"if that endpoint requires authentication; if the endpoint is a PMM Server, the {{.server_username}} and "+
			"{{.server_password}} placeholders keep each client's own PMM Server credentials")
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
