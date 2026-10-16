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

	// m serializes reads, so that concurrent readers of a dynamic secret share one set of credentials.
	m           sync.Mutex
	token       string
	tokenExpiry time.Time
	cache       map[string]cachedSecret
}

type cachedSecret struct {
	data    map[string]string
	err     error
	read    time.Time
	expires time.Time
	leased  bool
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

	data, resp, err := c.read(ctx, path)
	if err != nil {
		c.cache[path] = cachedSecret{err: err, read: now, expires: now.Add(errorCacheTTL)}
		return nil, err
	}

	ttl := c.cfg.EffectiveCacheTTL()
	leased := resp.LeaseID != "" && resp.LeaseDuration > 0
	if leased {
		ttl = time.Duration(resp.LeaseDuration) * time.Second * 2 / 3 //nolint:mnd
	}
	c.cache[path] = cachedSecret{data: data, read: now, expires: now.Add(ttl), leased: leased}

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

	// Whatever happens below, a cached value may no longer be what Vault has.
	delete(c.cache, path)

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

	return c.withToken(ctx, func(token string) error {
		return c.do(ctx, http.MethodPost, path, token, body, nil)
	})
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
