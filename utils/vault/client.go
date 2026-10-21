// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vault

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/percona/pmm/utils/tlsconfig"
)

const (
	requestTimeout  = 10 * time.Second
	maxResponseSize = 1 << 20

	// For errorCacheTTL a failed read is answered with its error before Vault is asked again,
	// so that many readers of an unreachable Vault fail fast instead of waiting for it one by one.
	errorCacheTTL = 5 * time.Second
)

// Client reads secrets from Vault. It is safe for concurrent use.
type Client struct {
	cfg  Config
	http *http.Client
	now  func() time.Time

	// azureMetadataURL and gcpMetadataURL are where the azure and gcp methods get the identity to log in with.
	azureMetadataURL string
	gcpMetadataURL   string
	// getenv returns the variables that the cloud sets for the identity, such as for an AKS workload identity.
	getenv func(string) string

	// l logs that a secret read before stands in for one Vault cannot be read from, see SetLogger.
	l Logger

	// m serializes reads, so that concurrent readers of a dynamic secret share one set of credentials.
	m           sync.Mutex
	token       string
	tokenExpiry time.Time
	cache       map[string]cachedSecret

	// down is the error with which Vault could not be reached at all, and retryAt when it is asked again: until
	// then, reads answer at once with the data read last, or with that error, rather than each waiting for Vault.
	down    error
	retryAt time.Time
}

type cachedSecret struct {
	data    map[string]string
	err     error
	read    time.Time
	expires time.Time
	leased  bool

	// last is the data read last, which stands in for the secret while Vault cannot be reached: read at
	// lastRead, and, for a secret with a lease, until lastUntil, when its credentials stop working.
	last      map[string]string
	lastRead  time.Time
	lastUntil time.Time
	// stale reports that data is last, standing in for the secret; staleLogged that this was logged for the secret
	// alone, rather than for all of them as Vault could not be reached at all.
	stale       bool
	staleLogged bool
}

// Logger logs what the client does without being asked, see SetLogger.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
}

// SetLogger sets where the client logs that the secrets read before stand in for those that Vault cannot be read
// from, and that Vault answers again. It must be called before the client is used.
func (c *Client) SetLogger(l Logger) {
	c.l = l
}

// unanswered reports whether err means that Vault could not be reached, or could not answer, rather than that it
// refused to: the data read last stands in for the secret then. A refusal, such as for a revoked policy or a
// deleted secret, takes effect. A read that its caller canceled is neither.
func unanswered(ctx context.Context, err error) bool {
	if errors.Is(ctx.Err(), context.Canceled) {
		return false
	}
	if re, ok := errors.AsType[*responseError](err); ok {
		return re.status >= http.StatusInternalServerError || re.status == http.StatusTooManyRequests
	}
	return unreachable(ctx, err)
}

// unreachable reports whether err means that Vault could not be reached at all, or is sealed, so that it cannot
// answer for any secret, rather than for the one read. Waiting for the caller's deadline counts.
func unreachable(ctx context.Context, err error) bool {
	if errors.Is(ctx.Err(), context.Canceled) {
		return false
	}
	if re, ok := errors.AsType[*responseError](err); ok {
		return re.status == http.StatusServiceUnavailable
	}
	_, ok := errors.AsType[*url.Error](err)
	return ok || errors.Is(err, context.DeadlineExceeded)
}

// readThrough reads the secret from Vault, unless Vault could not be reached at all a moment ago. It is asked again
// only after the cache TTL then, so that one read waits for it rather than every read, one after another.
// Must be called with c.m held.
func (c *Client) readThrough(ctx context.Context, path string, now time.Time) (map[string]string, *secretResponse, error) {
	if c.down != nil && now.Before(c.retryAt) {
		return nil, nil, fmt.Errorf("vault GET %s: not asked until %s, since it could not be reached: %w",
			path, c.retryAt.Format(time.RFC3339), c.down)
	}

	data, resp, err := c.read(ctx, path)
	_, answered := errors.AsType[*responseError](err)
	switch {
	case err != nil && unreachable(ctx, err):
		if c.down == nil && c.l != nil {
			c.l.Warnf("Vault cannot be reached, using the secrets read before until it can: %s.", err)
		}
		c.down, c.retryAt = err, now.Add(c.cfg.EffectiveCacheTTL())
	case c.down != nil && (err == nil || answered):
		c.down = nil
		if c.l != nil {
			c.l.Infof("Vault can be reached again.")
		}
	}
	return data, resp, err
}

type readSinceKey struct{}

// ReadSince returns a context in which the client answers from its cache only with the secrets it read at
// or after t, and reads the others from Vault again. A secret with a lease is still answered from the cache
// until it is about to expire: reading it again would issue new credentials. It serves to check the secrets
// for changes, reading each of them once per check.
func ReadSince(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, readSinceKey{}, t)
}

// fresh reports whether a cached secret may answer a read in ctx, see ReadSince.
func (s cachedSecret) fresh(ctx context.Context, now time.Time) bool {
	if !now.Before(s.expires) {
		return false
	}
	since, ok := ctx.Value(readSinceKey{}).(time.Time)
	return !ok || s.leased || !s.read.Before(since)
}

// New returns a Client for an enabled configuration.
func New(cfg Config) (*Client, error) {
	if !cfg.Enabled() {
		return nil, errors.New("vault address is not set")
	}

	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	tlsConfig := tlsconfig.Get()
	tlsConfig.InsecureSkipVerify = cfg.TLSSkipVerify
	if cfg.CACert != "" {
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read vault CA certificate: %w", err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates found in vault CA certificate %s", cfg.CACert)
		}
		tlsConfig.RootCAs = pool
	}

	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert
	transport.TLSClientConfig = tlsConfig

	return &Client{
		cfg:   cfg,
		http:  &http.Client{Transport: transport, Timeout: requestTimeout},
		now:   time.Now,
		cache: make(map[string]cachedSecret),

		azureMetadataURL: azureMetadataURL,
		gcpMetadataURL:   gcpMetadataURL,
		getenv:           os.Getenv,
	}, nil
}

// Read returns the data of the secret at the API path, e.g. secret/data/pmm/mysql for the secret
// pmm/mysql of the KV version 2 engine mounted at secret/. The KV version 2 envelope is unwrapped.
//
// A secret with a lease, such as database credentials, is reused until two thirds of the lease
// have passed, so every reader gets the same credentials until they are about to expire. Any other
// secret is reused for the cache TTL.
func (c *Client) Read(ctx context.Context, path string) (map[string]string, error) {
	path = strings.Trim(path, "/")

	c.m.Lock()
	defer c.m.Unlock()

	now := c.now()
	if s, ok := c.cache[path]; ok && s.fresh(ctx, now) {
		return s.data, s.err
	}

	prev := c.cache[path]
	data, resp, err := c.readThrough(ctx, path, now)
	if err != nil {
		if prev.last != nil && unanswered(ctx, err) && (prev.lastUntil.IsZero() || now.Before(prev.lastUntil)) {
			// Vault is asked again after errorCacheTTL, as after any other error.
			stale := prev
			stale.data, stale.err, stale.expires, stale.stale = prev.last, nil, now.Add(errorCacheTTL), true
			// Logged for this secret only if Vault can be reached, but cannot answer for it.
			if !prev.stale && c.down == nil && c.l != nil {
				c.l.Warnf("Vault cannot answer for the secret %s, using the one read at %s until it can: %s.",
					path, prev.lastRead.Format(time.RFC3339), err)
				stale.staleLogged = true
			}
			c.cache[path] = stale
			return prev.last, nil
		}

		c.cache[path] = cachedSecret{err: err, read: now, expires: now.Add(errorCacheTTL)}
		return nil, err
	}
	if prev.staleLogged && c.l != nil {
		c.l.Infof("Vault answers for the secret %s again.", path)
	}

	ttl := c.cfg.EffectiveCacheTTL()
	leased := resp.LeaseID != "" && resp.LeaseDuration > 0
	entry := cachedSecret{data: data, read: now, leased: leased, last: data, lastRead: now}
	if leased {
		lease := time.Duration(resp.LeaseDuration) * time.Second
		ttl = lease * 2 / 3 //nolint:mnd
		entry.lastUntil = now.Add(lease)
	}
	entry.expires = now.Add(ttl)
	c.cache[path] = entry

	return data, nil
}

// Get returns the value of key in the secret at path.
func (c *Client) Get(ctx context.Context, path, key string) (string, error) {
	data, err := c.Read(ctx, path)
	if err != nil {
		return "", err
	}

	v, ok := data[key]
	if !ok {
		return "", &KeyNotFoundError{Path: strings.Trim(path, "/"), Key: key}
	}
	return v, nil
}

// KeyNotFoundError is returned when a secret has no value for the key asked for.
type KeyNotFoundError struct {
	Path string
	Key  string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("vault secret %s has no key %s", e.Path, e.Key)
}

// GetRef returns the value a vault:<path>#<key> reference points to.
func (c *Client) GetRef(ctx context.Context, ref string) (string, error) {
	r, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	return c.Get(ctx, r.Path, r.Key)
}

type secretResponse struct {
	LeaseID       string         `json:"lease_id"`
	LeaseDuration int64          `json:"lease_duration"`
	Data          map[string]any `json:"data"`
}

type loginResponse struct {
	Auth *struct {
		ClientToken   string `json:"client_token"`
		LeaseDuration int64  `json:"lease_duration"`
	} `json:"auth"`
}

type responseError struct {
	method string
	path   string
	status int
	errors []string
}

func (e *responseError) Error() string {
	msg := fmt.Sprintf("vault %s %s: %d %s", e.method, e.path, e.status, http.StatusText(e.status))
	if len(e.errors) != 0 {
		msg += ": " + strings.Join(e.errors, "; ")
	}
	return msg
}

// Write stores data in the secret at the API path of a KV secrets engine, version 1 or 2, keeping
// the keys of the secret that data does not have. The path is the one Read takes.
func (c *Client) Write(ctx context.Context, path string, data map[string]string) error {
	path = strings.Trim(path, "/")

	c.m.Lock()
	defer c.m.Unlock()

	// Whatever happens below, a cached value may no longer be what Vault has, so it is read again. The data read
	// last is kept to stand in for it while Vault cannot be reached, unless the write below replaces it.
	cached := c.cache[path]
	c.cache[path] = cachedSecret{
		expires: time.Time{}, last: cached.last, lastRead: cached.lastRead, lastUntil: cached.lastUntil,
	}

	// Checked before reading: reading the path of another engine may have side effects, such as the
	// database engine creating a user.
	kv2, err := c.kvVersion2(ctx, path)
	if err != nil {
		return err
	}

	current, _, err := c.read(ctx, path)
	re, notFound := errors.AsType[*responseError](err)
	switch {
	case notFound && re.status == http.StatusNotFound, errors.Is(err, errNoData):
		current = nil
	case err != nil:
		return err
	}

	merged := maps.Clone(current)
	if merged == nil {
		merged = make(map[string]string, len(data))
	}
	maps.Copy(merged, data)

	var body any = merged
	if kv2 {
		body = map[string]any{"data": merged}
	}

	err = c.withToken(ctx, func(token string) error {
		return c.do(ctx, http.MethodPost, path, token, body, nil)
	})
	if err != nil {
		return err
	}

	// What Vault has now, should it not be reachable to read it again.
	c.cache[path] = cachedSecret{last: merged, lastRead: c.now()}
	return nil
}

// WriteRef stores value under the key a vault:<path>#<key> reference points to.
func (c *Client) WriteRef(ctx context.Context, ref, value string) error {
	r, err := ParseRef(ref)
	if err != nil {
		return err
	}
	return c.Write(ctx, r.Path, map[string]string{r.Key: value})
}

// kvVersion2 reports whether the path belongs to a KV version 2 engine, whose writes wrap the data, and
// returns an error for an engine other than KV. It asks Vault, which answers any token with a capability
// on the path. If Vault does not answer, only the API path of KV version 2, <mount>/data/<name>, is taken
// for KV: any other path may belong to an engine that reading has side effects for.
// Must be called with c.m held.
func (c *Client) kvVersion2(ctx context.Context, path string) (bool, error) {
	var resp struct {
		Data struct {
			Type    string            `json:"type"`
			Options map[string]string `json:"options"`
		} `json:"data"`
	}
	err := c.withToken(ctx, func(token string) error {
		return c.do(ctx, http.MethodGet, "sys/internal/ui/mounts/"+path, token, nil, &resp)
	})
	if err == nil && resp.Data.Type != "" {
		if resp.Data.Type != "kv" && resp.Data.Type != "generic" {
			return false, fmt.Errorf("vault path %s belongs to a %s secrets engine, only KV secrets can be written", path, resp.Data.Type)
		}
		return resp.Data.Options["version"] == "2", nil
	}

	segments := strings.Split(path, "/")
	if len(segments) > 2 && segments[1] == "data" {
		return true, nil
	}
	if err == nil {
		err = errors.New("no secrets engine in the answer")
	}
	return false, fmt.Errorf("cannot tell the secrets engine of vault path %s, only KV secrets can be written: %w", path, err)
}

// read reads the secret. Must be called with c.m held.
func (c *Client) read(ctx context.Context, path string) (map[string]string, *secretResponse, error) {
	var resp secretResponse
	err := c.withToken(ctx, func(token string) error {
		resp = secretResponse{}
		return c.do(ctx, http.MethodGet, path, token, nil, &resp)
	})
	if err != nil {
		return nil, nil, err
	}

	data, err := secretData(resp.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("vault secret %s: %w", path, err)
	}

	return data, &resp, nil
}

// withToken calls fn with a token, logging in again once if Vault rejects the token it has.
// Must be called with c.m held.
func (c *Client) withToken(ctx context.Context, fn func(token string) error) error {
	token, err := c.authToken(ctx)
	if err != nil {
		return err
	}

	err = fn(token)
	re, ok := errors.AsType[*responseError](err)
	if !ok || re.status != http.StatusForbidden {
		return err
	}

	c.token = ""
	token, err = c.authToken(ctx)
	if err != nil {
		return err
	}
	return fn(token)
}

// authToken returns a token, logging in when there is none or it is about to expire.
// Must be called with c.m held.
func (c *Client) authToken(ctx context.Context) (string, error) {
	if c.token != "" && (c.tokenExpiry.IsZero() || c.now().Before(c.tokenExpiry)) {
		return c.token, nil
	}

	token, ttl, err := c.login(ctx)
	if err != nil {
		return "", err
	}

	c.token = token
	c.tokenExpiry = time.Time{}
	if ttl > 0 {
		c.tokenExpiry = c.now().Add(ttl * 2 / 3) //nolint:mnd
	}
	return token, nil
}

func (c *Client) login(ctx context.Context) (string, time.Duration, error) {
	var body map[string]any
	switch c.cfg.authMethod() {
	case AuthMethodToken:
		if c.cfg.TokenFile == "" {
			return c.cfg.Token, 0, nil
		}
		token, err := readSecretFile(c.cfg.TokenFile)
		return token, 0, err

	case AuthMethodAppRole:
		secretID := c.cfg.SecretID
		if c.cfg.SecretIDFile != "" {
			var err error
			secretID, err = readSecretFile(c.cfg.SecretIDFile)
			if err != nil {
				return "", 0, err
			}
		}
		body = map[string]any{"role_id": c.cfg.RoleID, "secret_id": secretID}

	case AuthMethodKubernetes:
		jwtFile := c.cfg.JWTFile
		if jwtFile == "" {
			jwtFile = DefaultKubernetesJWTFile
		}
		jwt, err := readSecretFile(jwtFile)
		if err != nil {
			return "", 0, err
		}
		body = map[string]any{"role": c.cfg.Role, "jwt": jwt}

	case AuthMethodAWS:
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return "", 0, fmt.Errorf("failed to load AWS configuration: %w", err)
		}
		body, err = awsLoginData(ctx, awsCfg.Credentials, c.cfg.AWSRegion, c.cfg.AWSServerID, c.now())
		if err != nil {
			return "", 0, err
		}
		body["role"] = c.cfg.Role

	case AuthMethodAzure:
		resource := c.cfg.AzureResource
		if resource == "" {
			resource = DefaultAzureResource
		}
		var err error
		if azureWorkloadIdentity(c.getenv) {
			body, err = azureWorkloadIdentityLoginData(ctx, c.getenv, resource, c.cfg.AzureClientID)
		} else {
			body, err = azureLoginData(ctx, c.azureMetadataURL, resource, c.cfg.AzureClientID)
		}
		if err != nil {
			return "", 0, err
		}
		body["role"] = c.cfg.Role

	case AuthMethodGCP:
		var err error
		body, err = gcpLoginData(ctx, c.gcpMetadataURL, c.cfg.Role)
		if err != nil {
			return "", 0, err
		}
		body["role"] = c.cfg.Role
	}

	var resp loginResponse
	err := c.do(ctx, http.MethodPost, "auth/"+strings.Trim(c.cfg.authMount(), "/")+"/login", "", body, &resp)
	if err != nil {
		return "", 0, fmt.Errorf("vault login with method %s failed: %w", c.cfg.authMethod(), err)
	}
	if resp.Auth == nil || resp.Auth.ClientToken == "" {
		return "", 0, fmt.Errorf("vault login with method %s returned no token", c.cfg.authMethod())
	}

	return resp.Auth.ClientToken, time.Duration(resp.Auth.LeaseDuration) * time.Second, nil
}

func (c *Client) do(ctx context.Context, method, path, token string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	u := strings.TrimSuffix(c.cfg.Address, "/") + "/v1/" + escapePath(path)
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Request", "true")
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}
	if c.cfg.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.cfg.Namespace)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("vault %s %s: %w", method, path, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("vault %s %s: %w", method, path, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		var e struct {
			Errors []string `json:"errors"`
		}
		_ = json.Unmarshal(b, &e)
		return &responseError{method: method, path: path, status: resp.StatusCode, errors: e.Errors}
	}

	if out == nil || len(b) == 0 {
		return nil
	}

	err = json.Unmarshal(b, out)
	if err != nil {
		return fmt.Errorf("vault %s %s: invalid response: %w", method, path, err)
	}
	return nil
}

var errNoData = errors.New("the secret version has no data, it may be deleted or destroyed")

// secretData flattens the data of a secret, unwrapping the KV version 2 envelope.
func secretData(raw map[string]any) (map[string]string, error) {
	if _, ok := raw["metadata"]; ok {
		if _, ok := raw["data"]; ok {
			inner, _ := raw["data"].(map[string]any)
			if inner == nil {
				return nil, errNoData
			}
			raw = inner
		}
	}

	res := make(map[string]string, len(raw))
	for k, v := range raw {
		switch v := v.(type) {
		case string:
			res[k] = v
		case nil:
			res[k] = ""
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			res[k] = string(b)
		}
	}
	return res, nil
}

func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

func readSecretFile(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return s, nil
}
