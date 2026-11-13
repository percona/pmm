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

package envsecrets

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lib/pq"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/clickhouse"
	"github.com/percona/pmm/managed/services/supervisord"
	"github.com/percona/pmm/managed/utils/secrets"
)

const (
	// The directory where the entrypoint installs the certificate given by PMM_TLS_CERTIFICATE.
	nginxDir = "/srv/nginx"

	// The Unix socket directory of the built-in PostgreSQL, whose postgres superuser it trusts.
	postgresSocketDir = "/run/postgresql"

	// How long ClickHouse takes at most to read its changed users: it checks them every few seconds.
	clickHouseReloadWait = 5 * time.Second

	// The file with the password of PMM Extensions that the entrypoint generates when PMM_EXTENSIONS_POSTGRES_PASSWORD
	// is unset.
	extensionsPasswordFile = "/srv/.extensions_postgres_password" //nolint:gosec
)

// runStep applies st with the values the variables have now.
func (w *Watcher) runStep(ctx context.Context, st step) error {
	switch st {
	case installTLS:
		err := w.a.installTLS(w.getenv("PMM_TLS_CERTIFICATE"), w.getenv("PMM_TLS_CERTIFICATE_KEY"), w.getenv("PMM_TLS_CA_CERTIFICATE"))
		if err != nil {
			return err
		}
		return w.a.reloadNginx(ctx)

	case applyClickHousePasswords:
		if on, _ := strconv.ParseBool(w.getenv("PMM_DISABLE_BUILTIN_CLICKHOUSE")); on {
			return nil
		}
		err := w.a.applyClickHousePasswords(w.getenv)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.clickHouseWait):
			return nil
		}

	case setGrafanaDatabasePassword:
		// The role is the one of the built-in PostgreSQL, unless Grafana has a database elsewhere.
		if w.getenv("GF_DATABASE_URL") != "" || w.getenv("GF_DATABASE_HOST") != "" {
			return nil
		}
		if on, _ := strconv.ParseBool(w.getenv("PMM_DISABLE_BUILTIN_POSTGRES")); on {
			return nil
		}
		return w.a.setRolePassword(ctx, cmp.Or(w.getenv("GF_DATABASE_USER"), "grafana"), w.getenv("GF_DATABASE_PASSWORD"))

	case saveVMCredentials:
		return w.a.saveVMCredentials(cmp.Or(w.getenv("PMM_VM_URL"), models.VMBaseURL))

	case publishExtensions:
		return w.a.publishExtensions(ctx, w.getenv)

	case noStep:
	}
	return nil
}

// systemActions are the actions of a running PMM Server.
type systemActions struct {
	sup *supervisord.Service
}

func (systemActions) writeRotatedEnv(env map[string]string) error {
	return secrets.WriteRotatedEnv(secrets.RotatedEnvPath, env)
}

// installTLS installs the certificate and its key as the entrypoint does, and the CA certificate if given: one
// given in the environment rather than in Vault is not known, and the file the entrypoint installed is kept.
func (systemActions) installTLS(cert, key, ca string) error {
	if cert == "" || key == "" {
		return errors.New("PMM_TLS_CERTIFICATE and PMM_TLS_CERTIFICATE_KEY must be given together")
	}
	// A certificate and a key that do not belong together are not installed: nginx would not serve them. Rotated
	// one after the other, they are installed at the check after the second.
	_, err := tls.X509KeyPair([]byte(cert), []byte(key))
	if err != nil {
		return fmt.Errorf("the TLS certificate and key in Vault do not belong together: %w", err)
	}

	for _, f := range []struct {
		name, content string
		perm          os.FileMode
	}{
		{"certificate.key", key, 0o600},
		{"certificate.crt", cert, 0o644},
		{"ca-certs.pem", ca, 0o644},
	} {
		if f.content == "" {
			continue
		}
		err = writeFile(filepath.Join(nginxDir, f.name), strings.TrimRight(f.content, "\n")+"\n", f.perm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (systemActions) applyClickHousePasswords(getenv func(string) string) error {
	return clickhouse.ApplyPasswords(getenv)
}

// setRolePassword sets the password of a role of the built-in PostgreSQL, as its superuser over the socket.
func (systemActions) setRolePassword(ctx context.Context, role, password string) error {
	db, err := models.OpenDB(models.SetupDBParams{Address: postgresSocketDir, Username: "postgres"})
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER ROLE %s WITH PASSWORD %s", pq.QuoteIdentifier(role), pq.QuoteLiteral(password)))
	if err != nil {
		return fmt.Errorf("cannot set the password of the role %s: %w", role, err)
	}
	return nil
}

func (systemActions) saveVMCredentials(vmURL string) error {
	u, err := models.ParseVictoriaMetricsURL(vmURL)
	if err != nil {
		return err
	}
	return supervisord.SaveVMCredentials(u)
}

// publishExtensions sets the password of the role of PMM Extensions, and publishes it and the secret key to the
// side-car, with the scripts the entrypoint runs.
func (systemActions) publishExtensions(ctx context.Context, getenv func(string) string) error {
	password := getenv("PMM_EXTENSIONS_POSTGRES_PASSWORD")
	if password == "" {
		b, err := os.ReadFile(extensionsPasswordFile)
		if err != nil {
			return fmt.Errorf("the password of PMM Extensions given in the environment is not known, restart PMM Server to apply the change: %w", err)
		}
		password = strings.TrimSpace(string(b))
	}

	env := append(
		os.Environ(),
		"POSTGRES_DATA_DIR=/srv/postgres18",
		"POSTGRES_OLD_DATA_DIR=/srv/postgres14",
		"POSTGRES_PASSWORD_FILE=/srv/.postgres_password",
		"POSTGRES_BIN_DIR=/usr/pgsql-18/bin",
		"PMM_EXTENSIONS_POSTGRES_PASSWORD="+password,
	)
	if key := getenv("PMM_EXTENSIONS_SECRET_KEY"); key != "" {
		env = append(env, "PMM_EXTENSIONS_SECRET_KEY="+key)
	}

	ha, _ := strconv.ParseBool(getenv("PMM_HA_ENABLE"))
	external, _ := strconv.ParseBool(getenv("PMM_DISABLE_BUILTIN_POSTGRES"))
	scripts := []string{"/opt/ansible/roles/extensions/files/extensions-secrets"}
	if !ha && !external {
		scripts = append([]string{"/opt/ansible/roles/postgres/files/postgres-extensions"}, scripts...)
	}
	for _, script := range scripts {
		cmd := exec.CommandContext(ctx, "bash", script) //nolint:gosec // one of the scripts above
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %w: %s", filepath.Base(script), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func (a systemActions) restart(ctx context.Context, program string) error {
	return a.sup.RestartSupervisedService(ctx, program)
}

// reloadNginx has nginx read its configuration, and the certificate with it, again, without dropping connections.
func (a systemActions) reloadNginx(ctx context.Context) error {
	return a.sup.SignalSupervisedService(ctx, "nginx", "HUP")
}

// restartSelf stops pmm-managed as a SIGTERM does, after which supervisord starts it again. supervisorctl restart
// would not: it is a child of pmm-managed, and does not live to start it again.
func (systemActions) restartSelf() {
	_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
}

// writeFile replaces the file at path with one holding content, so that a reader never sees it half written.
func writeFile(path, content string, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	_, err = tmp.WriteString(content)
	if err == nil {
		err = tmp.Chmod(perm)
	}
	if err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
