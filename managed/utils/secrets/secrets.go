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

// Package secrets resolves PMM Server credentials to files under one canonical directory and reads them.
package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Dir is the canonical directory every PMM Server component reads its credentials from.
const Dir = "/run/pmm/secrets"

//nolint:gosec // a variable name, not a credential
const (
	// SecretsDirEnvVar names a directory with one file per credential, for example a mounted Kubernetes Secret.
	SecretsDirEnvVar = "PMM_SECRETS_DIR"

	// Subdirectory of the values PMM writes itself: legacy environment variables and defaults.
	envSubdir = ".env"
)

// Name identifies a credential; it is also its file name in Dir.
type Name string

// Credentials PMM Server resolves.
//
//nolint:gosec // file names, not credentials
const (
	PostgresPassword             Name = "postgres-password"
	GrafanaDBPassword            Name = "grafana-db-password"
	ClickHousePassword           Name = "clickhouse-password"
	ClickHouseDatasourcePassword Name = "clickhouse-datasource-password"
	AdminPassword                Name = "admin-password"
	GrafanaSecretKey             Name = "grafana-secret-key"
)

// Source tells where a resolved credential came from.
type Source string

// Credential sources, in resolution order.
const (
	SourceFile    Source = "file"
	SourceDir     Source = "secrets-dir"
	SourceEnv     Source = "environment"
	SourceDefault Source = "default"
	SourceUnset   Source = "unset"
)

// spec describes where one credential can come from.
type spec struct {
	name Name
	// fileEnvs name variables holding a path to the credential file.
	fileEnvs []string
	// valueEnvs name legacy variables holding the value itself.
	valueEnvs []string
	// providers are the Grafana providers that expanded a legacy value before PMM-15168, in Grafana's order. Grafana
	// doesn't expand what it reads from a file, so they apply here.
	providers []string
	// defaultValue is the value PMM has always used when nothing is configured.
	defaultValue string
}

// Grafana's variable providers (pkg/setting/expanders.go); ${NAME} is the short form of $__env{NAME}.
const (
	envProvider  = "__env"
	fileProvider = "__file"
)

// grafanaConfig is the order in which Grafana expands its configuration values: variables first, then files.
var grafanaConfig = []string{envProvider, fileProvider}

// grafanaVar is Grafana's pattern for a provider expression such as ${NAME} or $__file{path}.
var grafanaVar = regexp.MustCompile(`\$(|__\w+){([^}]+)}`)

var catalog = []spec{
	{
		name:         PostgresPassword,
		fileEnvs:     []string{"PMM_POSTGRES_PASSWORD_FILE"},
		valueEnvs:    []string{"PMM_POSTGRES_DBPASSWORD"},
		defaultValue: "pmm-managed",
	},
	{
		name:         GrafanaDBPassword,
		fileEnvs:     []string{"PMM_GRAFANA_DB_PASSWORD_FILE"},
		valueEnvs:    []string{"GF_DATABASE_PASSWORD"},
		providers:    grafanaConfig,
		defaultValue: "grafana",
	},
	{
		name:         ClickHousePassword,
		fileEnvs:     []string{"PMM_CLICKHOUSE_PASSWORD_FILE"},
		valueEnvs:    []string{"PMM_CLICKHOUSE_PASSWORD"},
		defaultValue: "clickhouse",
	},
	{
		name:      ClickHouseDatasourcePassword,
		fileEnvs:  []string{"PMM_CLICKHOUSE_DATASOURCE_PASSWORD_FILE"},
		valueEnvs: []string{"PMM_CLICKHOUSE_DATASOURCE_PASSWORD"},
		// Grafana's provisioning expanded ${PMM_CLICKHOUSE_DATASOURCE_PASSWORD} with its variables pass, so only
		// the files pass applied to the value.
		providers:    []string{fileProvider},
		defaultValue: "grafana",
	},
	{
		name:      AdminPassword,
		fileEnvs:  []string{"PMM_ADMIN_PASSWORD_FILE"},
		valueEnvs: []string{"PMM_ADMIN_PASSWORD", "GF_SECURITY_ADMIN_PASSWORD"},
		providers: grafanaConfig,
	},
	{
		name:      GrafanaSecretKey,
		fileEnvs:  []string{"PMM_GRAFANA_SECRET_KEY_FILE"},
		valueEnvs: []string{"GF_SECURITY_SECRET_KEY"},
		// Grafana's encryption read GF_SECURITY_SECRET_KEY again as written (SectionWithEnvOverrides), so expanding it
		// would change the key that existing secrets are encrypted with.
	},
}

// EnvVarNames returns the environment variables the catalog reads besides the legacy values, for the environment
// parser.
func EnvVarNames() []string {
	names := []string{SecretsDirEnvVar}
	for _, s := range catalog {
		names = append(names, s.fileEnvs...)
	}
	return names
}

// Path returns the canonical path of a credential.
func Path(name Name) string {
	return filepath.Join(Dir, string(name))
}

// ErrNotResolved means a credential has no link in Dir: it is unset, or an entrypoint that predates the secrets step
// started the program, as in the development container.
var ErrNotResolved = errors.New("credential not resolved")

// Exists reports whether a credential was resolved; one that cannot be read counts, so its reader fails loudly.
func Exists(name Name) bool {
	return exists(Path(name))
}

func exists(path string) bool {
	// A link whose file went away still counts.
	_, err := os.Lstat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

// Value returns value when set, otherwise the credential from its file. For a credential that was not resolved, it
// returns defaultValue and ErrNotResolved; any other error means that a resolved credential can't be read.
func Value(value string, name Name, defaultValue string) (string, error) {
	return valueIn(Dir, value, name, defaultValue)
}

func valueIn(dir, value string, name Name, defaultValue string) (string, error) {
	if value != "" {
		return value, nil
	}
	path := filepath.Join(dir, string(name))
	if !exists(path) {
		return defaultValue, ErrNotResolved
	}
	return readFile(path)
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // paths the operator configured or Dir
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return v, nil
}

// Resolved is the outcome for one credential. Path is empty when the credential is unset.
type Resolved struct {
	Name   Name
	Source Source
	Path   string
	// Var names the legacy variable of a value from the environment.
	Var string
	// Padded reports a value from the environment with leading or trailing whitespace, which every reader drops.
	Padded bool
}

// Options configures Prepare.
type Options struct {
	Getenv func(string) string
	// RunDir is the canonical directory, Dir in production.
	RunDir string
}

// OptionsFromEnv returns the production options.
func OptionsFromEnv() Options {
	return Options{
		Getenv: os.Getenv,
		RunDir: Dir,
	}
}

// Prepare resolves every credential to a file and links it at its canonical path.
//
// It returns all resolution problems at once, so a misconfigured installation is fixed in one round.
func Prepare(opts Options) ([]Resolved, error) {
	// Values written by an earlier start must not outlive the variables they came from.
	err := os.RemoveAll(filepath.Join(opts.RunDir, envSubdir))
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(opts.RunDir, 0o700) //nolint:mnd
	if err != nil {
		return nil, err
	}

	var errs []error
	resolved := make([]Resolved, 0, len(catalog))
	for _, s := range catalog {
		r, err := resolve(opts, s)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.name, err))
			continue
		}

		link := filepath.Join(opts.RunDir, string(s.name))
		err = os.Remove(link)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
			continue
		}
		if r.Path != "" {
			// A relative target would resolve against the link's directory, not where the file was found.
			target, err := filepath.Abs(r.Path)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", s.name, err))
				continue
			}
			err = os.Symlink(target, link)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
		resolved = append(resolved, r)
	}

	return resolved, errors.Join(errs...)
}

func resolve(opts Options, s spec) (Resolved, error) {
	r := Resolved{Name: s.name}
	file, _, err := single(opts.Getenv, s.fileEnvs)
	if err != nil {
		return r, err
	}
	value, key, err := single(opts.Getenv, s.valueEnvs)
	if err != nil {
		return r, err
	}
	if file != "" && value != "" {
		return r, fmt.Errorf("both a file (%s) and a value (%s) are set", strings.Join(s.fileEnvs, "/"), strings.Join(s.valueEnvs, "/"))
	}
	// Grafana read a value that is only $__file{path} from that file, so the file stays the source.
	if p := grafanaFile(value, s.providers); p != "" {
		file, value = p, ""
	}

	if file != "" {
		_, err = readFile(file)
		if err != nil {
			return r, err
		}
		r.Source, r.Path = SourceFile, file
		return r, nil
	}

	p, err := dirSource(opts, s, value)
	if err != nil {
		return r, err
	}
	if p != "" {
		r.Source, r.Path = SourceDir, p
		return r, nil
	}

	switch {
	case value != "":
		value, err = expand(opts.Getenv, value, s.providers)
		if err != nil {
			return r, fmt.Errorf("%s: %w", key, err)
		}
		if strings.TrimSpace(value) == "" {
			return r, fmt.Errorf("%s resolves to an empty value", key)
		}
		r.Source, r.Var, r.Padded = SourceEnv, key, strings.TrimSpace(value) != value
	case s.defaultValue != "":
		r.Source, value = SourceDefault, s.defaultValue
	default:
		r.Source = SourceUnset
		return r, nil
	}
	r.Path = filepath.Join(opts.RunDir, envSubdir, string(s.name))
	return r, writeFile(r.Path, value)
}

// dirSource returns the credential's file in PMM_SECRETS_DIR, or "" when there is none.
func dirSource(opts Options, s spec, value string) (string, error) {
	dir := opts.Getenv(SecretsDirEnvVar)
	if dir == "" {
		return "", nil
	}
	p := filepath.Join(dir, string(s.name))
	_, err := os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if value != "" {
		return "", fmt.Errorf("both %s and a value (%s) are set", p, strings.Join(s.valueEnvs, "/"))
	}
	_, err = readFile(p)
	if err != nil {
		return "", err
	}
	return p, nil
}

// single returns the one value set among the given variables and a variable that holds it; set variables must agree.
func single(getenv func(string) string, keys []string) (string, string, error) {
	var found, foundKey string
	for _, k := range keys {
		v := getenv(k)
		if v == "" {
			continue
		}
		if found != "" && v != found {
			return "", "", fmt.Errorf("%s and %s are set to different values", foundKey, k)
		}
		found, foundKey = v, k
	}
	return found, foundKey, nil
}

// grafanaFile returns the path of a value that is only $__file{path}, when the files provider applies to it.
func grafanaFile(value string, providers []string) string {
	if !slices.Contains(providers, fileProvider) {
		return ""
	}
	m := grafanaVar.FindStringSubmatch(value)
	if m == nil || m[0] != value || m[1] != fileProvider {
		return ""
	}
	return m[2]
}

// expand applies Grafana's providers to a value the way Grafana does: one provider after the other over the whole
// value, so a variable may hold a $__file{}, and providers Grafana doesn't have are left as they are.
func expand(getenv func(string) string, value string, providers []string) (string, error) {
	for _, p := range providers {
		for _, m := range grafanaVar.FindAllStringSubmatch(value, -1) {
			provider := m[1]
			if provider == "" {
				provider = envProvider
			}
			if provider != p {
				continue
			}

			var v string
			switch p {
			case envProvider:
				v = getenv(m[2])
			case fileProvider:
				b, err := os.ReadFile(m[2]) //nolint:gosec // a path the operator configured
				if err != nil {
					return "", err
				}
				v = strings.TrimSpace(string(b))
			}
			value = strings.Replace(value, m[0], v, 1)
		}
	}
	return value, nil
}

// writeFile writes a credential atomically, readable by its owner only, without a trailing newline.
func writeFile(path, value string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0o700) //nolint:mnd
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) //nolint:errcheck

	_, err = f.WriteString(value)
	if err != nil {
		_ = f.Close()
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
