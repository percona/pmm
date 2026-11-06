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

// Package envsecrets applies the secrets of PMM Server given as Vault references that are rotated in Vault while it
// runs, without a restart of PMM Server.
package envsecrets

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/services/supervisord"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

// refEnvPrefix is the prefix of the variables that keep the references of the variables vault-env resolved.
const refEnvPrefix = "PMM_VAULT_REF_"

// pmmManaged is the program of pmm-managed, which restarts itself last.
const pmmManaged = "pmm-managed"

// actions are what applying a rotated secret takes, see systemActions.
type actions interface {
	writeRotatedEnv(env map[string]string) error
	installTLS(cert, key, ca string) error
	applyClickHousePasswords(getenv func(string) string) error
	setRolePassword(ctx context.Context, role, password string) error
	saveVMCredentials(vmURL string) error
	publishExtensions(ctx context.Context, getenv func(string) string) error
	restart(ctx context.Context, program string) error
	reloadNginx(ctx context.Context) error
	restartSelf()
}

// step applies what the programs do not read from their environment, or what they need first.
type step int

const (
	noStep step = iota
	installTLS
	applyClickHousePasswords
	setGrafanaDatabasePassword
	saveVMCredentials
	publishExtensions
)

// change is how a variable rotated in Vault is applied.
type change struct {
	// env is set for a variable that the programs read from their environment: its new value goes to the
	// rotated variables that they start with.
	env  bool
	step step
	// restart are the programs that read it at start, pmmManaged included.
	restart []string
	// note is logged once it is applied.
	note string
}

// changeOf returns how the variable name is applied, and false for one that cannot be.
func changeOf(name string) (change, bool) {
	switch name {
	case "GF_SECURITY_ADMIN_PASSWORD":
		// Applied by grafana.AdminPassword.
		return change{}, false
	case "GF_SECURITY_SECRET_KEY":
		// Grafana cannot read the secrets it stored with another key.
		return change{}, false
	case "PMM_TLS_CERTIFICATE", "PMM_TLS_CERTIFICATE_KEY", "PMM_TLS_CA_CERTIFICATE":
		return change{step: installTLS}, true
	case "PMM_CLICKHOUSE_PASSWORD":
		return change{env: true, step: applyClickHousePasswords, restart: []string{"qan-api2", pmmManaged}}, true
	case "PMM_CLICKHOUSE_DATASOURCE_PASSWORD":
		return change{env: true, step: applyClickHousePasswords, restart: []string{"grafana"}}, true
	case "GF_DATABASE_PASSWORD":
		return change{env: true, step: setGrafanaDatabasePassword, restart: []string{"grafana", pmmManaged}}, true
	case "PMM_POSTGRES_DBPASSWORD":
		// pmm-managed sets the password of its role in the built-in PostgreSQL itself, when it cannot log in.
		return change{env: true, restart: []string{"grafana", pmmManaged}}, true
	case "PMM_VM_URL":
		return change{env: true, step: saveVMCredentials, restart: []string{"vmproxy", pmmManaged}}, true
	case "PMM_EXTENSIONS_POSTGRES_PASSWORD", "PMM_EXTENSIONS_SECRET_KEY":
		return change{step: publishExtensions, note: "Restart the PMM Extensions side-car to apply it."}, true
	}

	switch {
	case strings.HasPrefix(name, "GF_"):
		return change{env: true, restart: []string{"grafana"}}, true
	case strings.HasPrefix(name, "VM_"):
		return change{env: true, restart: []string{"victoriametrics"}}, true
	case strings.HasPrefix(name, "VMAGENT_"), strings.HasPrefix(name, "PMM_"):
		// The vmagents get theirs from pmm-managed, which reads the other PMM_* variables too.
		return change{env: true, restart: []string{pmmManaged}}, true
	}
	return change{}, false
}

// Watcher applies the variables of PMM Server given as Vault references when they are rotated in Vault.
type Watcher struct {
	refs  map[string]string
	known map[string]string
	// env is the environment PMM Server started with, for the variables that are not references.
	env  map[string]string
	read func(ctx context.Context, ref string) (string, error)
	a    actions
	l    *logrus.Entry

	// rotated are the values of the variables with env changes rotated so far, see secrets.RotatedEnvPath.
	rotated map[string]string
	// clickHouseWait is how long to wait for ClickHouse to read its changed users.
	clickHouseWait time.Duration

	// keyRef is the reference of the encryption key, which PMM Server reads itself, rather than vault-env. It cannot
	// be applied while PMM Server runs, nor at all without its data, so a change is reported instead, see checkKey.
	keyRef string
	// keyInUse returns the fingerprint of the encryption key that PMM Server uses.
	keyInUse func() (string, error)
	// keyReported is the fingerprint of the key in Vault that was reported as not the one in use.
	keyReported string
}

// New returns a Watcher of the variables of environ that vault-env resolved, which restarts the programs with sup.
func New(environ []string, sup *supervisord.Service) *Watcher {
	w := &Watcher{
		refs:  make(map[string]string),
		known: make(map[string]string),
		read: func(ctx context.Context, ref string) (string, error) {
			// Read from Vault, not from the cache, so that a change is seen at this check.
			return secrets.Resolve(vault.ReadSince(ctx, time.Now()), ref)
		},
		a:       systemActions{sup: sup},
		l:       logrus.WithField("component", "env-secrets"),
		rotated: make(map[string]string),

		clickHouseWait: clickHouseReloadWait,
		keyInUse:       encryption.Fingerprint,
	}

	env := make(map[string]string)
	for _, kv := range environ {
		name, value, _ := strings.Cut(kv, "=")
		env[name] = value
	}
	w.env = env
	if ref := env[encryption.KeyEnvVar]; vault.IsRef(ref) {
		w.keyRef = ref
	}
	for name, value := range env {
		if target, ok := strings.CutPrefix(name, refEnvPrefix); ok && vault.IsRef(value) {
			w.refs[target] = value
			if v, set := env[target]; set {
				// What PMM Server runs with; those that the entrypoint unsets are read at the first check.
				w.known[target] = v
			}
		}
	}

	// The variables rotated before pmm-managed restarted, which the programs restarted since run with.
	rotated, err := secrets.ReadRotatedEnv(secrets.RotatedEnvPath)
	if err != nil {
		w.l.Warnf("Cannot read the secrets rotated in Vault before: %s.", err)
	}
	maps.Copy(w.rotated, rotated)
	return w
}

// Run checks the variables for changes in Vault at every interval, and applies them, until ctx is done.
func (w *Watcher) Run(ctx context.Context, interval time.Duration) {
	if len(w.refs) == 0 && w.keyRef == "" {
		return
	}

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		w.check(ctx)

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// checkKey reports, once for each, an encryption key in Vault that is not the one PMM Server uses. The secrets that
// PMM Server stores are encrypted with the one it uses: it keeps using that one, and would not start with another.
func (w *Watcher) checkKey(ctx context.Context) {
	if w.keyRef == "" {
		return
	}

	key, err := w.read(ctx, w.keyRef)
	if err != nil {
		w.l.Warnf("Cannot check %s for a change in Vault: %s.", encryption.KeyEnvVar, err)
		return
	}
	inVault, err := encryption.KeyFingerprint(key)
	if err != nil {
		inVault = "invalid: " + err.Error()
	}
	inUse, err := w.keyInUse()
	if err != nil {
		w.l.Warnf("Cannot check %s for a change in Vault: %s.", encryption.KeyEnvVar, err)
		return
	}

	switch {
	case inVault == inUse:
		if w.keyReported != "" {
			w.l.Infof("%s in Vault is the key that PMM Server uses again.", encryption.KeyEnvVar)
		}
		w.keyReported = ""
	case inVault != w.keyReported:
		w.l.Errorf("%s in Vault is not the key that PMM Server uses, and encrypted the secrets it stores with: "+
			"PMM Server keeps using the one it has, and would not start with the one in Vault. Put the key back in "+
			"Vault, or change it with pmm-encryption-rotation, which encrypts the secrets with the new one.", encryption.KeyEnvVar)
		w.keyReported = inVault
	}
}

// check reads the variables from Vault, and applies those that changed.
func (w *Watcher) check(ctx context.Context) {
	w.checkKey(ctx)

	changed := make(map[string]change)
	for _, name := range slices.Sorted(maps.Keys(w.refs)) {
		value, err := w.read(ctx, w.refs[name])
		if err != nil {
			w.l.Warnf("Cannot check %s for a change in Vault: %s.", name, err)
			continue
		}

		known, ok := w.known[name]
		if !ok {
			w.known[name] = value
			continue
		}
		if value == known {
			continue
		}

		c, applicable := changeOf(name)
		if applicable && c.env && w.rotated[name] == value {
			// Applied already, but not to pmm-managed, which would restart itself again and again otherwise.
			w.l.Errorf("%s changed in Vault, but pmm-managed was not started with it, restart PMM Server to apply it.", name)
			w.known[name] = value
			continue
		}
		if !applicable {
			if name != "GF_SECURITY_ADMIN_PASSWORD" {
				w.l.Warnf("%s changed in Vault, which PMM Server cannot apply while it runs.", name)
			}
			w.known[name] = value
			continue
		}
		w.l.Infof("%s changed in Vault, applying it.", name)
		changed[name] = c
		w.known[name] = value
	}
	if len(changed) == 0 {
		return
	}

	err := w.apply(ctx, changed)
	if err != nil {
		w.l.Errorf("Cannot apply the secrets changed in Vault: %s.", err)
		// Applied again at the next check.
		for name := range changed {
			delete(w.known, name)
		}
	}
}

// apply applies the changed variables: their values go to the programs, which are restarted, pmm-managed last.
func (w *Watcher) apply(ctx context.Context, changed map[string]change) error {
	envChanged := false
	for name, c := range changed {
		if c.env {
			w.rotated[name] = w.known[name]
			envChanged = true
		}
	}
	if envChanged {
		err := w.a.writeRotatedEnv(w.rotated)
		if err != nil {
			return err
		}
	}

	steps := make(map[step]struct{})
	restart := make(map[string]struct{})
	var notes []string
	for _, name := range slices.Sorted(maps.Keys(changed)) {
		c := changed[name]
		if c.step != noStep {
			steps[c.step] = struct{}{}
		}
		for _, p := range c.restart {
			restart[p] = struct{}{}
		}
		if c.note != "" && !slices.Contains(notes, c.note) {
			notes = append(notes, c.note)
		}
	}

	for _, st := range slices.Sorted(maps.Keys(steps)) {
		err := w.runStep(ctx, st)
		if err != nil {
			return err
		}
	}

	for _, p := range slices.Sorted(maps.Keys(restart)) {
		if p == pmmManaged {
			continue
		}
		w.l.Infof("Restarting %s to apply the secrets changed in Vault.", p)
		err := w.a.restart(ctx, p)
		if err != nil {
			return err
		}
	}
	for _, note := range notes {
		w.l.Info(note)
	}
	if _, ok := restart[pmmManaged]; ok {
		w.l.Info("Restarting pmm-managed to apply the secrets changed in Vault.")
		w.a.restartSelf()
	}
	return nil
}

// getenv returns the value of the variable name that PMM Server has now: rotated in Vault, or the one it started with.
func (w *Watcher) getenv(name string) string {
	if v, ok := w.known[name]; ok {
		return v
	}
	return w.env[name]
}
