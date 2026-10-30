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
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/encryption"
)

// fakeActions records the actions taken.
type fakeActions struct {
	rotatedEnv map[string]string
	tls        []string
	clickHouse []string
	roles      map[string]string
	vmURL      string
	extensions int
	restarted  []string
	nginx      int
	self       int
	installErr error
}

func (f *fakeActions) writeRotatedEnv(env map[string]string) error {
	f.rotatedEnv = maps.Clone(env)
	return nil
}

func (f *fakeActions) installTLS(cert, key, ca string) error {
	if f.installErr != nil {
		return f.installErr
	}
	f.tls = []string{cert, key, ca}
	return nil
}

func (f *fakeActions) applyClickHousePasswords(getenv func(string) string) error {
	f.clickHouse = []string{getenv("PMM_CLICKHOUSE_PASSWORD"), getenv("PMM_CLICKHOUSE_DATASOURCE_PASSWORD")}
	return nil
}

func (f *fakeActions) setRolePassword(_ context.Context, role, password string) error {
	if f.roles == nil {
		f.roles = make(map[string]string)
	}
	f.roles[role] = password
	return nil
}

func (f *fakeActions) saveVMCredentials(vmURL string) error {
	f.vmURL = vmURL
	return nil
}

func (f *fakeActions) publishExtensions(context.Context, func(string) string) error {
	f.extensions++
	return nil
}

func (f *fakeActions) restart(_ context.Context, program string) error {
	f.restarted = append(f.restarted, program)
	return nil
}

func (f *fakeActions) reloadNginx(context.Context) error {
	f.nginx++
	return nil
}

func (f *fakeActions) restartSelf() {
	f.self++
}

// newTestWatcher returns a Watcher of environ, whose references point to the secrets of vault, by reference.
func newTestWatcher(t *testing.T, environ []string, vault map[string]string) (*Watcher, *fakeActions) {
	t.Helper()

	w := New(environ, nil)
	w.rotated = make(map[string]string)
	f := &fakeActions{}
	w.a = f
	w.clickHouseWait = 0
	w.read = func(_ context.Context, ref string) (string, error) {
		v, ok := vault[ref]
		if !ok {
			return "", errors.New("permission denied")
		}
		return v, nil
	}
	return w, f
}

func TestWatcher(t *testing.T) {
	t.Parallel()

	t.Run("a Grafana setting", func(t *testing.T) {
		t.Parallel()

		vault := map[string]string{"vault:pmm#smtp": "old"}
		w, f := newTestWatcher(t, []string{"GF_SMTP_PASSWORD=old", "PMM_VAULT_REF_GF_SMTP_PASSWORD=vault:pmm#smtp"}, vault)

		w.check(t.Context())
		assert.Empty(t, f.restarted, "not changed")

		vault["vault:pmm#smtp"] = "new"
		w.check(t.Context())
		assert.Equal(t, map[string]string{"GF_SMTP_PASSWORD": "new"}, f.rotatedEnv)
		assert.Equal(t, []string{"grafana"}, f.restarted)
		assert.Zero(t, f.self)

		w.check(t.Context())
		assert.Len(t, f.restarted, 1, "applied once")
	})

	t.Run("the ClickHouse password", func(t *testing.T) {
		t.Parallel()

		vault := map[string]string{"vault:pmm#ch": "old"}
		w, f := newTestWatcher(t, []string{
			"PMM_CLICKHOUSE_PASSWORD=old", "PMM_VAULT_REF_PMM_CLICKHOUSE_PASSWORD=vault:pmm#ch",
			"PMM_CLICKHOUSE_DATASOURCE_PASSWORD=ds",
		}, vault)

		vault["vault:pmm#ch"] = "new"
		w.check(t.Context())
		assert.Equal(t, []string{"new", "ds"}, f.clickHouse)
		assert.Equal(t, map[string]string{"PMM_CLICKHOUSE_PASSWORD": "new"}, f.rotatedEnv)
		assert.Equal(t, []string{"qan-api2"}, f.restarted)
		assert.Equal(t, 1, f.self, "pmm-managed restarts itself last")
	})

	t.Run("the password of the Grafana database", func(t *testing.T) {
		t.Parallel()

		vault := map[string]string{"vault:pmm#gf": "old"}
		environ := []string{"GF_DATABASE_PASSWORD=old", "PMM_VAULT_REF_GF_DATABASE_PASSWORD=vault:pmm#gf"}
		w, f := newTestWatcher(t, environ, vault)
		vault["vault:pmm#gf"] = "new"
		w.check(t.Context())
		assert.Equal(t, map[string]string{"grafana": "new"}, f.roles)
		assert.Equal(t, []string{"grafana"}, f.restarted)
		assert.Equal(t, 1, f.self)

		// A database elsewhere is not PMM Server's to change.
		vault["vault:pmm#gf"] = "old"
		w, f = newTestWatcher(t, append(environ, "GF_DATABASE_HOST=db.example.com"), vault)
		vault["vault:pmm#gf"] = "newer"
		w.check(t.Context())
		assert.Empty(t, f.roles)
		assert.Equal(t, []string{"grafana"}, f.restarted)
	})

	t.Run("the TLS certificate", func(t *testing.T) {
		t.Parallel()

		// The entrypoint unsets them, so what PMM Server has is read at the first check.
		vault := map[string]string{"vault:pmm#crt": "c1", "vault:pmm#key": "k1"}
		w, f := newTestWatcher(t, []string{
			"PMM_VAULT_REF_PMM_TLS_CERTIFICATE=vault:pmm#crt", "PMM_VAULT_REF_PMM_TLS_CERTIFICATE_KEY=vault:pmm#key",
		}, vault)
		w.check(t.Context())
		assert.Nil(t, f.tls)

		// Rotated one after the other: not installed until they belong together.
		vault["vault:pmm#crt"] = "c2"
		f.installErr = errors.New("the TLS certificate and key in Vault do not belong together")
		w.check(t.Context())
		assert.Nil(t, f.tls)
		assert.Zero(t, f.nginx)

		vault["vault:pmm#key"] = "k2"
		f.installErr = nil
		w.check(t.Context())
		assert.Equal(t, []string{"c2", "k2", ""}, f.tls)
		assert.Equal(t, 1, f.nginx)
		assert.Nil(t, f.rotatedEnv, "not in the environment of any program")
		assert.Empty(t, f.restarted)
		assert.Zero(t, f.self)
	})

	t.Run("several at once", func(t *testing.T) {
		t.Parallel()

		vault := map[string]string{"vault:pmm#vm": "http://u:p1@vm:8428/", "vault:pmm#pg": "p1", "vault:pmm#ext": "e1"}
		w, f := newTestWatcher(t, []string{
			"PMM_VM_URL=http://u:p1@vm:8428/", "PMM_VAULT_REF_PMM_VM_URL=vault:pmm#vm",
			"PMM_POSTGRES_DBPASSWORD=p1", "PMM_VAULT_REF_PMM_POSTGRES_DBPASSWORD=vault:pmm#pg",
			"PMM_VAULT_REF_PMM_EXTENSIONS_SECRET_KEY=vault:pmm#ext",
		}, vault)
		w.check(t.Context())

		vault["vault:pmm#vm"] = "http://u:p2@vm:8428/"
		vault["vault:pmm#pg"] = "p2"
		vault["vault:pmm#ext"] = "e2"
		w.check(t.Context())
		assert.Equal(t, "http://u:p2@vm:8428/", f.vmURL)
		assert.Equal(t, 1, f.extensions)
		assert.Equal(t, map[string]string{"PMM_VM_URL": "http://u:p2@vm:8428/", "PMM_POSTGRES_DBPASSWORD": "p2"}, f.rotatedEnv)
		assert.Equal(t, []string{"grafana", "vmproxy"}, f.restarted, "each once")
		assert.Equal(t, 1, f.self)
	})

	t.Run("not applied while PMM Server runs", func(t *testing.T) {
		t.Parallel()

		vault := map[string]string{"vault:pmm#key": "k1", "vault:pmm#admin": "a1"}
		w, f := newTestWatcher(t, []string{
			"GF_SECURITY_SECRET_KEY=k1", "PMM_VAULT_REF_GF_SECURITY_SECRET_KEY=vault:pmm#key",
			"GF_SECURITY_ADMIN_PASSWORD=a1", "PMM_VAULT_REF_GF_SECURITY_ADMIN_PASSWORD=vault:pmm#admin",
		}, vault)
		vault["vault:pmm#key"] = "k2"
		vault["vault:pmm#admin"] = "a2"
		w.check(t.Context())
		assert.Nil(t, f.rotatedEnv)
		assert.Empty(t, f.restarted)
		assert.Zero(t, f.self)
	})

	t.Run("pmm-managed started without it", func(t *testing.T) {
		t.Parallel()

		vault := map[string]string{"vault:pmm#agent": "new"}
		w, f := newTestWatcher(t, []string{
			"VMAGENT_remoteWrite_basicAuth_password=old",
			"PMM_VAULT_REF_VMAGENT_remoteWrite_basicAuth_password=vault:pmm#agent",
		}, vault)
		w.rotated["VMAGENT_remoteWrite_basicAuth_password"] = "new"
		w.check(t.Context())
		assert.Zero(t, f.self, "not restarted again and again")
	})

	t.Run("nothing from Vault", func(t *testing.T) {
		t.Parallel()

		w, _ := newTestWatcher(t, []string{"GF_SMTP_PASSWORD=plain"}, nil)
		require.Empty(t, w.refs)
		w.Run(t.Context(), 0)
	})
}

func TestWatcherEncryptionKey(t *testing.T) {
	t.Parallel()

	keys := make([]string, 3)
	for i := range keys {
		var err error
		keys[i], err = (&encryption.Encryption{}).GenerateKey()
		require.NoError(t, err)
	}
	inUse, err := encryption.KeyFingerprint(keys[0])
	require.NoError(t, err)

	vault := map[string]string{"vault:pmm#key": keys[0]}
	w, f := newTestWatcher(t, []string{"PMM_ENCRYPTION_KEY=vault:pmm#key"}, vault)
	w.keyInUse = func() (string, error) { return inUse, nil }
	logger, hook := logtest.NewNullLogger()
	w.l = logrus.NewEntry(logger)
	errorsLogged := func() int {
		n := 0
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.ErrorLevel {
				n++
			}
		}
		return n
	}

	w.check(t.Context())
	assert.Zero(t, errorsLogged())

	// Changed in Vault by hand: reported once, and never applied.
	vault["vault:pmm#key"] = keys[1]
	w.check(t.Context())
	w.check(t.Context())
	assert.Equal(t, 1, errorsLogged())
	assert.Contains(t, hook.LastEntry().Message, "pmm-encryption-rotation")
	assert.Empty(t, f.restarted)
	assert.Zero(t, f.self)

	vault["vault:pmm#key"] = keys[2]
	w.check(t.Context())
	assert.Equal(t, 2, errorsLogged(), "another key is reported again")

	vault["vault:pmm#key"] = keys[0]
	w.check(t.Context())
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	assert.Contains(t, hook.LastEntry().Message, "is the key that PMM Server uses again")
}
