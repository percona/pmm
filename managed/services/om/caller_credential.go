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

package om

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/utils/auth"
)

// sessionCookie is login_cookie_name in PMM's grafana.ini.
const sessionCookie = "pmm_session"

// exchangeRenewAfter is the share of an exchanged bearer's lifetime after which it is renewed, so it never expires in flight.
const exchangeRenewAfter = 0.8

type serviceCallKey struct{}

// asService marks the PMM Extensions calls made with ctx as pmm-managed's own work, made with the shared token.
func asService(ctx context.Context) context.Context {
	return context.WithValue(ctx, serviceCallKey{}, true)
}

type credential struct {
	bearer string
	// session is the caller's PMM session the bearer was exchanged for, if it was.
	session string
	caller  bool
}

type exchangedBearer struct {
	token   string
	renewAt time.Time
}

// credential returns what a PMM Extensions call made with ctx authenticates with: the shared token only for a call marked asService.
func (c *extensionsClient) credential(ctx context.Context) (credential, error) {
	if marked, _ := ctx.Value(serviceCallKey{}).(bool); marked {
		return credential{bearer: c.token}, nil
	}

	headers, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return credential{}, status.Error(codes.Unauthenticated, "the request carries no credential for PMM Extensions")
	}

	if authorization := headers.Get("Authorization"); authorization != "" {
		token := auth.GetTokenFromHeaders(headers)
		switch {
		case token != "":
			return credential{bearer: token, caller: true}, nil
		case strings.HasPrefix(authorization, "Basic"):
			return credential{}, status.Error(codes.Unauthenticated,
				"PMM Extensions does not accept a username and password; authenticate with a service account token instead")
		default:
			return credential{}, status.Error(codes.Unauthenticated, "the Authorization header holds no credential PMM Extensions accepts")
		}
	}

	cookie, err := (&http.Request{Header: headers}).Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return credential{}, status.Error(codes.Unauthenticated, "the request carries no PMM session for PMM Extensions")
	}
	bearer, err := c.exchangeSession(ctx, cookie.Value)
	if err != nil {
		return credential{}, err
	}
	return credential{bearer: bearer, session: cookie.Value, caller: true}, nil
}

func (c *extensionsClient) exchangeSession(ctx context.Context, session string) (string, error) {
	key := sha256.Sum256([]byte(session))
	c.exchangedMu.Lock()
	cached, ok := c.exchanged[key]
	c.exchangedMu.Unlock()
	if ok && time.Now().Before(cached.renewAt) {
		return cached.token, nil
	}

	// One exchange per session at a time, which a caller that gives up does not cancel for the others.
	minted := c.exchanges.DoChan(string(key[:]), func() (any, error) {
		return c.mintBearer(context.WithoutCancel(ctx), key, session)
	})
	select {
	case <-ctx.Done():
		return "", status.FromContextError(ctx.Err()).Err()
	case result := <-minted:
		if result.Err != nil {
			return "", result.Err
		}
		token, _ := result.Val.(string)
		return token, nil
	}
}

func (c *extensionsClient) mintBearer(ctx context.Context, key [sha256.Size]byte, session string) (string, error) {
	exchange := extensionsApp{client: c, path: "api/oauth/session"}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, exchange.endpoint("exchange"), nil)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to build the session exchange: %s", err)
	}
	req.Header.Set("Cookie", sessionCookie+"="+session)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", status.Errorf(codes.Unavailable, "PMM Extensions is unreachable: %s", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return "", status.Error(codes.Unauthenticated, "PMM Extensions did not accept the PMM session")
	default:
		return "", status.Errorf(codes.Internal,
			"PMM Extensions answered %s to the session exchange: %s", resp.Status, extensionsErrorDetail(resp))
	}

	var minted struct {
		AccessToken string  `json:"access_token"`
		ExpiresIn   float64 `json:"expires_in"`
	}
	err = json.NewDecoder(resp.Body).Decode(&minted)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to decode the session exchange: %s", err)
	}
	if minted.AccessToken == "" {
		return "", status.Error(codes.Internal, "PMM Extensions returned no bearer for the PMM session")
	}

	now := time.Now()
	c.exchangedMu.Lock()
	defer c.exchangedMu.Unlock()
	if c.exchanged == nil {
		c.exchanged = make(map[[sha256.Size]byte]exchangedBearer)
	}
	for k, e := range c.exchanged {
		if !now.Before(e.renewAt) {
			delete(c.exchanged, k)
		}
	}
	renewAt := now.Add(time.Duration(minted.ExpiresIn * exchangeRenewAfter * float64(time.Second)))
	c.exchanged[key] = exchangedBearer{token: minted.AccessToken, renewAt: renewAt}
	return minted.AccessToken, nil
}

// forget drops the bearer exchanged for session, so the next call exchanges it again.
func (c *extensionsClient) forget(session string) {
	key := sha256.Sum256([]byte(session))
	c.exchangedMu.Lock()
	delete(c.exchanged, key)
	c.exchangedMu.Unlock()
}
