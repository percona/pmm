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
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	omv1 "github.com/percona/pmm/api/om/v1"
)

const emptyHostsPage = `{"items": [], "total": 0, "offset": 0, "limit": 200}`

// callerContext is a gRPC request's context, carrying a service account bearer unless pairs say otherwise.
func callerContext(t *testing.T, pairs ...string) context.Context {
	t.Helper()

	if len(pairs) == 0 {
		pairs = []string{"authorization", "Bearer glsa_caller"}
	}
	return metadata.NewIncomingContext(t.Context(), metadata.Pairs(pairs...))
}

func basic(credentials string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

func newHostsStub(t *testing.T, appCode, exchangeCode int) *extensionsStub {
	t.Helper()

	return newExtensionsStub(t, exchangeCode, []int{appCode}, []string{emptyHostsPage})
}

func TestCallerCredentialIsForwarded(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		service       bool
		pairs         []string
		authorization string
	}{
		{
			name:          "service work uses the shared token",
			service:       true,
			authorization: "Bearer test-token",
		},
		{
			name:          "a service account bearer is passed through",
			pairs:         []string{"authorization", "Bearer glsa_user"},
			authorization: "Bearer glsa_user",
		},
		{
			name:          "a service token sent as Basic is passed on as a bearer",
			pairs:         []string{"authorization", basic("service_token:glsa_user")},
			authorization: "Bearer glsa_user",
		},
		{
			name:          "the PMM session is exchanged for a bearer",
			pairs:         []string{"grpcgateway-cookie", "pmm_session=session-1"},
			authorization: "Bearer minted",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := newHostsStub(t, http.StatusOK, http.StatusOK)
			ctx := asService(t.Context())
			if !tc.service {
				ctx = callerContext(t, tc.pairs...)
			}

			_, err := stub.service(t).ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})

			require.NoError(t, err)
			require.Len(t, stub.calls, 1)
			assert.Equal(t, tc.authorization, stub.calls[0].authorization)
		})
	}
}

func TestCallerWithoutUsableCredentialIsRefused(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		pairs   []string
		message string
	}{
		{
			name:    "a call that is neither a request nor marked as service work",
			message: "the request carries no credential for PMM Extensions",
		},
		{
			name:    "a username and password",
			pairs:   []string{"authorization", basic("admin:admin")},
			message: "PMM Extensions does not accept a username and password; authenticate with a service account token instead",
		},
		{
			name:    "an empty bearer",
			pairs:   []string{"authorization", "Bearer "},
			message: "the Authorization header holds no credential PMM Extensions accepts",
		},
		{
			name:    "no PMM session among the cookies",
			pairs:   []string{"grpcgateway-cookie", "other=1"},
			message: "the request carries no PMM session for PMM Extensions",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := newHostsStub(t, http.StatusOK, http.StatusOK)
			ctx := t.Context()
			if tc.pairs != nil {
				ctx = callerContext(t, tc.pairs...)
			}

			_, err := stub.service(t).ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})

			require.Error(t, err)
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
			assert.Equal(t, tc.message, status.Convert(err).Message())
			assert.Empty(t, stub.calls, "nothing may reach PMM Extensions, least of all with the shared token")
		})
	}
}

func TestSessionExchange(t *testing.T) {
	t.Parallel()

	t.Run("only the PMM session cookie is sent", func(t *testing.T) {
		t.Parallel()

		stub := newHostsStub(t, http.StatusOK, http.StatusOK)
		ctx := callerContext(t, "grpcgateway-cookie", "grafana_session_expiry=1; pmm_session=session-1")

		_, err := stub.service(t).ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		require.Len(t, stub.exchanges, 1)
		assert.Equal(t, "pmm_session=session-1", stub.exchanges[0].cookie)
		assert.Empty(t, stub.exchanges[0].authorization)
	})

	t.Run("a minted bearer is reused until it is due for renewal", func(t *testing.T) {
		t.Parallel()

		stub := newHostsStub(t, http.StatusOK, http.StatusOK)
		svc := stub.service(t)
		ctx := callerContext(t, "grpcgateway-cookie", "pmm_session=session-1")

		for range 2 {
			_, err := svc.ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})
			require.NoError(t, err)
		}
		assert.Len(t, stub.exchanges, 1)

		client := svc.probe.app.client
		key := sha256.Sum256([]byte("session-1"))
		client.exchangedMu.Lock()
		cached := client.exchanged[key]
		assert.WithinDuration(t, time.Now().Add(240*time.Second), cached.renewAt, 5*time.Second,
			"renewal is due at four fifths of the 300s lifetime")
		cached.renewAt = time.Now().Add(-time.Second)
		client.exchanged[key] = cached
		client.exchangedMu.Unlock()

		_, err := svc.ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		assert.Len(t, stub.exchanges, 2)
	})

	t.Run("concurrent calls for one session share an exchange", func(t *testing.T) {
		t.Parallel()

		stub := newHostsStub(t, http.StatusOK, http.StatusOK)
		svc := stub.service(t)
		ctx := callerContext(t, "grpcgateway-cookie", "pmm_session=session-1")

		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				_, err := svc.ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})
				assert.NoError(t, err)
			})
		}
		wg.Wait()

		assert.Len(t, stub.exchanges, 1)
		assert.Len(t, stub.calls, 5)
	})

	t.Run("a caller that gives up does not cancel the exchange for the others", func(t *testing.T) {
		t.Parallel()

		release := make(chan struct{})
		var exchanges atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == stubExchangePath {
				exchanges.Add(1)
				<-release
				_, _ = w.Write([]byte(`{"access_token": "minted", "expires_in": 300}`))
				return
			}
			_, _ = w.Write([]byte(emptyHostsPage))
		}))
		t.Cleanup(server.Close)
		svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(server.URL, "test-token")
		ctx := callerContext(t, "grpcgateway-cookie", "pmm_session=session-1")

		quitting, quit := context.WithCancel(ctx)
		gaveUp := make(chan error, 1)
		go func() {
			_, err := svc.ListInventoryHosts(quitting, &omv1.ListInventoryHostsRequest{})
			gaveUp <- err
		}()
		require.Eventually(t, func() bool { return exchanges.Load() == 1 }, 5*time.Second, 10*time.Millisecond)
		staying := make(chan error, 1)
		go func() {
			_, err := svc.ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})
			staying <- err
		}()

		quit()
		assert.Equal(t, codes.Canceled, status.Code(<-gaveUp))
		close(release)
		require.NoError(t, <-staying)
		assert.Equal(t, int32(1), exchanges.Load())
	})

	t.Run("the probe and bootstrap apps share exchanged bearers", func(t *testing.T) {
		t.Parallel()

		stub := newExtensionsStub(t, http.StatusOK, []int{http.StatusOK}, []string{emptyHostsPage, `[]`})
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")
		ctx := callerContext(t, "grpcgateway-cookie", "pmm_session=session-1")

		_, err := svc.ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)
		_, err = svc.ListBootstrapRuns(ctx, &omv1.ListBootstrapRunsRequest{})
		require.NoError(t, err)

		assert.Len(t, stub.exchanges, 1)
	})

	t.Run("a refused session reaches the caller as unauthenticated", func(t *testing.T) {
		t.Parallel()

		stub := newHostsStub(t, http.StatusOK, http.StatusUnauthorized)
		ctx := callerContext(t, "grpcgateway-cookie", "pmm_session=session-1")

		_, err := stub.service(t).ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})

		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Empty(t, stub.calls, "a refused exchange must not fall back to the shared token")
	})
}

func TestRefusalOfTheCaller(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		appCode int
		code    codes.Code
	}{
		{name: "a forbidden caller is told so", appCode: http.StatusForbidden, code: codes.PermissionDenied},
		{name: "a rejected bearer reads as unauthenticated", appCode: http.StatusUnauthorized, code: codes.Unauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := newHostsStub(t, tc.appCode, http.StatusOK)
			svc := stub.service(t)
			ctx := callerContext(t, "grpcgateway-cookie", "pmm_session=session-1")

			for range 2 {
				_, err := svc.ListInventoryHosts(ctx, &omv1.ListInventoryHostsRequest{})
				require.Error(t, err)
				assert.Equal(t, tc.code, status.Code(err))
			}
			assert.Len(t, stub.exchanges, 2, "a refusal must drop the exchanged bearer, so a changed role applies next time")
		})
	}

	t.Run("a rejected shared token stays PMM's own failure", func(t *testing.T) {
		t.Parallel()

		stub := newHostsStub(t, http.StatusForbidden, http.StatusOK)

		_, err := stub.service(t).ListInventoryHosts(asService(t.Context()), &omv1.ListInventoryHostsRequest{})

		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestInventoryEnabledSyncUsesTheSharedTokenInsideARequest(t *testing.T) {
	t.Parallel()

	// A ChangeSettings request carries an admin's credential, but the ENABLED route admits only the shared token.
	stub := newSEPStub(t, http.StatusOK, `{}`)

	svc := stub.service(t)
	svc.db = switchDB(t, false)
	svc.SyncInventoryEnabled(callerContext(t, "authorization", "Bearer glsa_admin"))

	require.Len(t, stub.calls, 1)
	assert.Equal(t, "/api/apps/om_inventory/enabled", stub.calls[0].path)
	assert.Equal(t, "Bearer test-token", stub.calls[0].authorization)
}
