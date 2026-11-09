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

	"github.com/percona/pmm/managed/utils/dir"
	"github.com/percona/pmm/utils/secretfile"
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

const (
	dirPerm  = os.FileMode(0o700)
	filePerm = os.FileMode(0o600)
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
	// valueEnvs name legacy variables holding the value itself; the first one set wins.
	valueEnvs []string
	// providers are the Grafana providers applied to a legacy value, in Grafana's order, since Grafana doesn't expand a file's content.
	providers []string
	// padding decides what a legacy value with leading or trailing whitespace does.
	padding padding
	// defaultValue is the value PMM has always used when nothing is configured.
	defaultValue string
}

// padding is how a legacy value with leading or trailing whitespace is handled, since every file reader drops it:
// trimmed with a warning, refused because it reached a database as written, or kept in its variable with no file,
// because Grafana reads it as written.
type padding int

const (
	trimPadding padding = iota
	refusePadding
	keepPadding
)

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
		padding:      refusePadding,
		defaultValue: "pmm-managed",
	},
	{
		name:         GrafanaDBPassword,
		fileEnvs:     []string{"PMM_GRAFANA_DB_PASSWORD_FILE"},
		valueEnvs:    []string{"GF_DATABASE_PASSWORD"},
		providers:    grafanaConfig,
		padding:      refusePadding,
		defaultValue: "grafana",
	},
	{
		name:         ClickHousePassword,
		fileEnvs:     []string{"PMM_CLICKHOUSE_PASSWORD_FILE"},
		valueEnvs:    []string{"PMM_CLICKHOUSE_PASSWORD"},
		padding:      refusePadding,
		defaultValue: "clickhouse",
	},
	{
		name:      ClickHouseDatasourcePassword,
		fileEnvs:  []string{"PMM_CLICKHOUSE_DATASOURCE_PASSWORD_FILE"},
		valueEnvs: []string{"PMM_CLICKHOUSE_DATASOURCE_PASSWORD"},
		// Grafana's provisioning expanded ${PMM_CLICKHOUSE_DATASOURCE_PASSWORD} with its variables pass, so only
		// the files pass applied to the value.
		providers:    []string{fileProvider},
		padding:      refusePadding,
		defaultValue: "grafana",
	},
	{
		name:     AdminPassword,
		fileEnvs: []string{"PMM_ADMIN_PASSWORD_FILE"},
		// Grafana read only GF_SECURITY_ADMIN_PASSWORD, so it wins over a different PMM_ADMIN_PASSWORD.
		valueEnvs: []string{"GF_SECURITY_ADMIN_PASSWORD", "PMM_ADMIN_PASSWORD"},
		providers: grafanaConfig,
		// Grafana sets it only when it creates its database, so a trimmed value changes nothing that exists.
		padding: trimPadding,
	},
	{
		name:      GrafanaSecretKey,
		fileEnvs:  []string{"PMM_GRAFANA_SECRET_KEY_FILE"},
		valueEnvs: []string{"GF_SECURITY_SECRET_KEY"},
		// Grafana's encryption read GF_SECURITY_SECRET_KEY again as written (SectionWithEnvOverrides), so expanding it
		// would change the key that existing secrets are encrypted with.
		padding: keepPadding,
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

// Value returns value when set, otherwise the credential from its file, or defaultValue and ErrNotResolved for a
// credential that was not resolved.
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
	return secretfile.Read(path)
}

// Resolved is the outcome for one credential; Path is empty when the credential is unset or Kept.
type Resolved struct {
	Name   Name
	Source Source
	Path   string
	// Var names the legacy variable that supplied the value.
	Var string
	// Padded reports a value from the environment with leading or trailing whitespace, which the file readers drop.
	Padded bool
	// Kept reports a padded value left in its variable with no file, because Grafana reads it as written.
	Kept bool
	// Ignored names a legacy variable set to a different value than Var, which wins.
	Ignored string
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

// Prepare resolves every credential to a file and links it at its canonical path, reporting all problems at once.
func Prepare(opts Options) ([]Resolved, error) {
	err := checkSecretsDir(opts)
	if err != nil {
		return nil, err
	}
	err = checkRunDir(opts)
	if err != nil {
		return nil, err
	}

	// Also makes a directory that already exists private, such as a tmpfs mounted there.
	err = dir.CreateDataDir(opts.RunDir, dirPerm)
	if err != nil {
		return nil, fmt.Errorf("%s must belong to the PMM Server user: %w", opts.RunDir, err)
	}
	// Values written by an earlier start must not outlive the variables they came from. .env itself stays, so a start
	// killed meanwhile leaves the earlier links next to it, where checkRunDir accepts them.
	envDir := filepath.Join(opts.RunDir, envSubdir)
	err = dir.CreateDataDir(envDir, dirPerm)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(envDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		err = os.Remove(filepath.Join(envDir, e.Name()))
		if err != nil {
			return nil, err
		}
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

	// A source that leads back through the links would make a loop that only the readers notice.
	for _, r := range resolved {
		if r.Path == "" {
			continue
		}
		_, err = secretfile.Read(filepath.Join(opts.RunDir, string(r.Name)))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.Name, err))
		}
	}

	return resolved, errors.Join(errs...)
}

func resolve(opts Options, s spec) (Resolved, error) {
	r := Resolved{Name: s.name}
	file, _, _ := first(opts.Getenv, s.fileEnvs)
	value, key, ignored := first(opts.Getenv, s.valueEnvs)
	r.Var, r.Ignored = key, ignored
	if file != "" && value != "" {
		return r, fmt.Errorf("both a file (%s) and a value (%s) are set", strings.Join(s.fileEnvs, "/"), strings.Join(s.valueEnvs, "/"))
	}
	// Grafana read a value that is only $__file{path} from that file, so the file stays the source.
	if p := grafanaFile(value, s.providers); p != "" {
		// It is still a value, so a file in PMM_SECRETS_DIR conflicts with it.
		_, err := dirSource(opts, s, value)
		if err != nil {
			return r, err
		}
		file, value = p, ""
	}

	if file != "" {
		// After a restart, the path would lead to the link from the earlier start, which becomes a link to itself.
		if inside(opts.RunDir, file) {
			return r, fmt.Errorf("%s is inside %s, where PMM Server links the credentials", file, opts.RunDir)
		}
		_, err := secretfile.Read(file)
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
		r.Source, r.Padded = SourceEnv, strings.TrimSpace(value) != value
		// Before the empty check: Grafana read the key as written, even one that is only whitespace.
		if r.Padded && s.padding == keepPadding {
			r.Kept = true
			return r, nil
		}
		if strings.TrimSpace(value) == "" {
			return r, fmt.Errorf("%s resolves to an empty value", key)
		}
		if r.Padded && s.padding == refusePadding {
			return r, fmt.Errorf("%s has leading or trailing whitespace, which PMM Server doesn't pass to the database: remove it", key)
		}
	case s.defaultValue != "":
		r.Source, value = SourceDefault, s.defaultValue
	default:
		r.Source = SourceUnset
		return r, nil
	}
	r.Path = filepath.Join(opts.RunDir, envSubdir, string(s.name))
	return r, dir.WriteFileAtomic(r.Path, []byte(value), filePerm)
}

// dirSource returns the credential's file in PMM_SECRETS_DIR, or "" when there is none.
func dirSource(opts Options, s spec, value string) (string, error) {
	dir := opts.Getenv(SecretsDirEnvVar)
	if dir == "" {
		return "", nil
	}
	p := filepath.Join(dir, string(s.name))
	// A link whose file is missing is a file that can't be read, not a missing one.
	_, err := os.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if value != "" {
		return "", fmt.Errorf("both %s and a value (%s) are set", p, strings.Join(s.valueEnvs, "/"))
	}
	_, err = secretfile.Read(p)
	if err != nil {
		return "", err
	}
	return p, nil
}

// first returns the value of the first variable set among keys, that variable, and a later one set to a different value.
func first(getenv func(string) string, keys []string) (string, string, string) {
	var value, key, differs string
	for _, k := range keys {
		v := getenv(k)
		switch {
		case v == "":
		case key == "":
			value, key = v, k
		case v != value && differs == "":
			differs = k
		}
	}
	return value, key, differs
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

// checkSecretsDir returns an error when PMM_SECRETS_DIR is set but isn't a directory outside the run directory.
func checkSecretsDir(opts Options) error {
	dir := opts.Getenv(SecretsDirEnvVar)
	if dir == "" {
		return nil
	}
	errInside := fmt.Errorf("%s (%s) is inside %s, where PMM Server links the credentials", SecretsDirEnvVar, dir, opts.RunDir)
	if inside(opts.RunDir, dir) {
		return errInside
	}
	// Only a missing file in the directory moves on to the next source; a missing directory is a mistake, such as a
	// volume that didn't mount.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("%s: %w", SecretsDirEnvVar, err)
	}
	// Followed through links too, because Prepare empties .env and replaces the links before it reads the directory.
	runDir, err := filepath.EvalSymlinks(opts.RunDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err == nil && inside(runDir, resolved) {
		return errInside
	}
	fi, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("%s: %w", SecretsDirEnvVar, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s (%s) is not a directory", SecretsDirEnvVar, dir)
	}
	return nil
}

// checkRunDir returns an error when the run directory holds anything that Prepare doesn't create, such as credentials
// mounted there, which Prepare would remove or replace.
func checkRunDir(opts Options) error {
	entries, err := os.ReadDir(opts.RunDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var foreign, links []string
	var env bool
	for _, e := range entries {
		switch {
		case e.Name() == envSubdir && e.IsDir():
			env = true
			files, err := os.ReadDir(filepath.Join(opts.RunDir, envSubdir))
			if err != nil {
				return err
			}
			for _, f := range files {
				// dir.WriteFileAtomic leaves .<name>.tmp<n> behind only when it is killed.
				if !f.Type().IsRegular() || !slices.ContainsFunc(catalog, func(s spec) bool {
					return f.Name() == string(s.name) || strings.HasPrefix(f.Name(), "."+string(s.name)+".tmp")
				}) {
					foreign = append(foreign, filepath.Join(envSubdir, f.Name()))
				}
			}
		case e.Type()&fs.ModeSymlink != 0 && slices.ContainsFunc(catalog, func(s spec) bool { return string(s.name) == e.Name() }):
			links = append(links, e.Name())
		default:
			foreign = append(foreign, e.Name())
		}
	}
	// Prepare creates .env at every start, so links without it are someone else's.
	if !env {
		foreign = append(foreign, links...)
	}
	if len(foreign) != 0 {
		return fmt.Errorf("%s holds %s, which PMM Server didn't create; mount credentials in another directory",
			opts.RunDir, strings.Join(foreign, ", "))
	}
	return nil
}

// inside reports whether path is dir or lies under it.
func inside(dir, path string) bool {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
