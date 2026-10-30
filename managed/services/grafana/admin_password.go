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

package grafana

import (
	"bytes"
	"cmp"
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

// adminUserID is the ID of the admin user that Grafana creates with GF_SECURITY_ADMIN_PASSWORD, whose password
// "grafana cli admin reset-admin-password" resets.
const adminUserID = 1

// postgresSocketDir is the directory of the Unix socket of the built-in PostgreSQL, whose superuser it trusts.
const postgresSocketDir = "/run/postgresql"

// maxPasswordChangeBody limits the body of a password change, which holds a few short fields.
const maxPasswordChangeBody = 64 << 10

// adminPasswordRetryInterval is how soon a failed check of the admin password is tried again.
const adminPasswordRetryInterval = 10 * time.Second

// passwordChangePathRE matches the paths of the password changes of Grafana: of the user's own, and of
// another user's by a server admin.
var passwordChangePathRE = regexp.MustCompile(`^/graph/api/(?:user|admin/users/(\d+))/password$`)

// AdminPassword keeps the password of the Grafana admin user the one in Vault, while GF_SECURITY_ADMIN_PASSWORD
// is a Vault reference. A password changed in Vault is applied to Grafana at the next check, without a
// restart, and a password changed through Grafana is stored in Vault before it takes effect: nginx passes the
// password changes of Grafana through ServeHTTP.
type AdminPassword struct {
	ref     string
	login   string
	grafana *url.URL
	http    *http.Client
	proxy   *httputil.ReverseProxy
	l       *logrus.Entry

	read  func(ctx context.Context) (string, error)
	write func(ctx context.Context, password string) error
	reset func(ctx context.Context, password string) error
	// stored returns the hash and the salt of the password of the admin user, as Grafana stores them, or is nil
	// where they cannot be read: from a database of Grafana elsewhere than the built-in PostgreSQL.
	stored func(ctx context.Context) (string, string, error)

	// m serializes the checks and the password changes. applied is the fingerprint of the password that
	// Grafana has, as far as known: empty until it is checked.
	m       sync.Mutex
	applied string
}

// NewAdminPassword returns an AdminPassword for the admin user login of the Grafana at addr, whose password
// the Vault reference ref gives. Without a reference, it only passes the password changes on to Grafana. The
// database of Grafana is the built-in PostgreSQL unless it is elsewhere.
func NewAdminPassword(ref, login, addr string, databaseElsewhere bool) *AdminPassword {
	grafana := &url.URL{Scheme: "http", Host: addr}
	a := &AdminPassword{
		ref:     ref,
		login:   login,
		grafana: grafana,
		http:    &http.Client{Timeout: time.Minute},
		proxy:   httputil.NewSingleHostReverseProxy(grafana),
		l:       logrus.WithField("component", "grafana/admin-password"),
		reset:   resetAdminPassword,
	}
	a.read = func(ctx context.Context) (string, error) {
		// Read from Vault, not from the cache, so that a change is applied at this check.
		return secrets.Resolve(vault.ReadSince(ctx, time.Now()), ref)
	}
	a.write = func(ctx context.Context, password string) error {
		return secrets.WriteRef(ctx, ref, password)
	}
	if !databaseElsewhere {
		a.stored = storedAdminPassword
	}
	return a
}

// fromVault reports whether the admin password is kept in Vault.
func (a *AdminPassword) fromVault() bool {
	return a.ref != ""
}

func fingerprint(password string) string {
	sum := sha256.Sum256([]byte(password))
	return string(sum[:])
}

// Run applies the admin password in Vault to Grafana, at once and then at every interval, until ctx is done.
func (a *AdminPassword) Run(ctx context.Context, interval time.Duration) {
	if !a.fromVault() {
		return
	}

	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		wait := interval
		err := a.sync(ctx)
		if err != nil {
			// Sooner, as at start, before Grafana is up.
			wait = min(interval, adminPasswordRetryInterval)
			a.l.Warnf("Cannot keep the admin password of Grafana the one in Vault, retrying in %s: %s.", wait, err)
		}
		t.Reset(wait)
	}
}

// sync applies the admin password in Vault to Grafana if it is not the one Grafana has. That is checked against
// the hash that Grafana stores, which catches a password changed by any means, without a login. Where the hash
// cannot be read, it is checked with Grafana at first, which uses up one of the failed logins that Grafana allows
// if it is not the one, and then it is known from what was applied since.
func (a *AdminPassword) sync(ctx context.Context) error {
	password, err := a.read(ctx)
	if err != nil {
		return err
	}

	a.m.Lock()
	defer a.m.Unlock()

	fp := fingerprint(password)
	if a.stored != nil {
		hash, salt, err := a.stored(ctx)
		if err == nil {
			if grafanaPasswordHash(password, salt) == hash {
				a.applied = fp
				return nil
			}
			if fp == a.applied {
				a.l.Warn("The admin password of Grafana was changed outside PMM, setting the one in Vault again.")
			}
			return a.apply(ctx, password)
		}
		a.l.Debugf("Cannot read the admin password that Grafana stores, checking it with Grafana: %s.", err)
	}

	if fp == a.applied {
		return nil
	}
	if a.applied == "" {
		matches, err := a.check(ctx, password)
		if err != nil {
			return err
		}
		if matches {
			a.applied = fp
			return nil
		}
	}

	return a.apply(ctx, password)
}

// apply sets the admin password of Grafana to password. Must be called with a.m held.
func (a *AdminPassword) apply(ctx context.Context, password string) error {
	err := a.reset(ctx, password)
	if err != nil {
		return err
	}
	a.applied = fingerprint(password)
	a.l.Info("Applied the admin password in Vault to Grafana.")
	return nil
}

// grafanaPasswordHash returns the hash that Grafana stores for password with salt.
func grafanaPasswordHash(password, salt string) string {
	key, err := pbkdf2.Key(sha256.New, password, []byte(salt), 10000, 50) //nolint:mnd // as Grafana does
	if err != nil {
		return ""
	}
	return hex.EncodeToString(key)
}

// storedAdminPassword reads the hash and the salt of the password of the admin user from the database of Grafana in
// the built-in PostgreSQL, as its superuser over the socket.
func storedAdminPassword(ctx context.Context) (string, string, error) {
	db, err := models.OpenDB(models.SetupDBParams{
		Address:  postgresSocketDir,
		Username: "postgres",
		Name:     cmp.Or(os.Getenv("GF_DATABASE_NAME"), "grafana"),
	})
	if err != nil {
		return "", "", err
	}
	defer db.Close() //nolint:errcheck

	var hash, salt string
	err = db.QueryRowContext(ctx, `SELECT password, salt FROM "user" WHERE id = $1`, adminUserID).Scan(&hash, &salt)
	if err != nil {
		return "", "", err
	}
	return hash, salt, nil
}

// check reports whether Grafana accepts password for the admin user.
func (a *AdminPassword) check(ctx context.Context, password string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.grafana.JoinPath("/api/user").String(), nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(a.login, password)

	resp, err := a.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("cannot check the admin password with Grafana: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusUnauthorized:
		return false, nil
	default:
		return false, fmt.Errorf("cannot check the admin password with Grafana: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
}

// ServeHTTP passes a password change on to Grafana. When it changes the password of the admin user, kept in
// Vault, the new password is stored in Vault once Grafana took it; if it cannot be, Grafana gets the password
// it had back, and the change fails. If the password in Vault cannot be read first, the change fails at once.
func (a *AdminPassword) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	m := passwordChangePathRE.FindStringSubmatch(req.URL.Path)
	if !a.fromVault() || m == nil || req.Method != http.MethodPut {
		a.proxy.ServeHTTP(rw, req)
		return
	}

	userID, err := a.targetUserID(req, m[1])
	if err != nil || userID != adminUserID {
		// Grafana answers as it would anyway: an unauthenticated or forbidden change fails there too.
		a.proxy.ServeHTTP(rw, req)
		return
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, maxPasswordChangeBody))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	var change struct {
		NewPassword string `json:"newPassword"` // the user's own
		Password    string `json:"password"`    // another user's, by a server admin
	}
	_ = json.Unmarshal(body, &change)
	password := change.NewPassword
	if m[1] != "" {
		password = change.Password
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	a.m.Lock()
	defer a.m.Unlock()

	// Read before Grafana takes the new one, to give it back if the new one cannot be stored in Vault.
	current, err := a.read(req.Context())
	if err != nil {
		a.l.Errorf("Cannot change the admin password: %s.", err)
		writeJSONError(rw, http.StatusServiceUnavailable,
			"The password cannot be changed while the one in Vault cannot be read: "+err.Error())
		return
	}

	rec := &recordedResponse{header: make(http.Header), status: http.StatusOK}
	a.proxy.ServeHTTP(rec, req)
	if rec.status == http.StatusOK && password != "" {
		err = a.write(req.Context(), password)
		if err != nil {
			a.l.Errorf("Cannot store the admin password changed in Grafana in Vault: %s.", err)
			a.restore(req.Context(), current)
			writeJSONError(rw, http.StatusInternalServerError,
				"The password could not be stored in Vault, so it was not changed: "+err.Error())
			return
		}
		a.applied = fingerprint(password)
		a.l.Info("Stored the admin password changed in Grafana in Vault.")
	}
	rec.copyTo(rw)
}

// restore gives Grafana back the admin password it had, after a new one it took could not be stored in Vault.
// Must be called with a.m held.
func (a *AdminPassword) restore(ctx context.Context, password string) {
	err := a.reset(ctx, password)
	if err != nil {
		// Checked with Grafana again at the next sync, which applies the password in Vault.
		a.applied = ""
		a.l.Errorf("Cannot give Grafana the admin password back: %s.", err)
		return
	}
	a.applied = fingerprint(password)
}

// targetUserID returns the ID of the user whose password the request changes: of the user who makes it, or the
// one in its path.
func (a *AdminPassword) targetUserID(req *http.Request, pathID string) (int, error) {
	path := "/api/user"
	if pathID != "" {
		path = "/api/users/" + pathID
	}

	// The address is Grafana's, and only the ID of the user, digits, comes from the request.
	r, err := http.NewRequestWithContext(req.Context(), http.MethodGet, a.grafana.JoinPath(path).String(), nil) //nolint:gosec
	if err != nil {
		return 0, err
	}
	for _, h := range []string{"Authorization", "Cookie", "X-Grafana-Org-Id"} {
		if v := req.Header.Values(h); len(v) != 0 {
			r.Header[h] = v
		}
	}

	resp, err := a.http.Do(r) //nolint:gosec
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	var user struct {
		ID int `json:"id"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, maxPasswordChangeBody)).Decode(&user)
	return user.ID, err
}

// resetAdminPassword sets the password of the admin user of Grafana with its command-line tool, which needs no
// password of a Grafana user to do so. The password is passed on stdin, to keep it out of the process list.
func resetAdminPassword(ctx context.Context, password string) error {
	cmd := exec.CommandContext(ctx, "grafana", "cli",
		"--config=/etc/grafana/grafana.ini", "--homepath=/usr/share/grafana",
		"admin", "reset-admin-password", "--password-from-stdin")
	cmd.Stdin = strings.NewReader(password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot reset the admin password of Grafana: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// recordedResponse records a response, so that it can be replaced before it is sent.
type recordedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recordedResponse) Header() http.Header         { return r.header }
func (r *recordedResponse) Write(b []byte) (int, error) { return r.body.Write(b) }
func (r *recordedResponse) WriteHeader(status int)      { r.status = status }

func (r *recordedResponse) copyTo(rw http.ResponseWriter) {
	maps.Copy(rw.Header(), r.header)
	rw.WriteHeader(r.status)
	_, _ = rw.Write(r.body.Bytes())
}

// writeJSONError writes an error in the format of Grafana's, which its web interface shows.
func writeJSONError(rw http.ResponseWriter, status int, message string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	err := json.NewEncoder(rw).Encode(map[string]string{"message": message})
	if err != nil {
		logrus.WithField("component", "grafana/admin-password").Debugf("Cannot write the error: %s.", err)
	}
}
