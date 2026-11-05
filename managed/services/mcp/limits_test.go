// Copyright (C) 2026 Percona LLC
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

package mcp

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiter(t *testing.T) {
	t.Parallel()

	l := newLimiter(2, 3)
	relA1, hit1 := l.acquire("a")
	relA2, hit2 := l.acquire("a")
	_, hit3 := l.acquire("a")
	require.Equal(t, limitNone, hit1)
	require.Equal(t, limitNone, hit2)
	require.Equal(t, limitPerCaller, hit3)
	relB, hitB := l.acquire("b")
	_, hitC := l.acquire("c")
	require.Equal(t, limitNone, hitB)
	require.Equal(t, limitPerReplica, hitC)
	// Both caps are full: the per-caller cap is reported.
	_, hitA := l.acquire("a")
	require.Equal(t, limitPerCaller, hitA)
	relA1()
	relA1()
	relA3, hitA3 := l.acquire("a")
	require.Equal(t, limitNone, hitA3)
	relA2()
	relA3()
	relB()
	require.Empty(t, l.byKey)
	require.Zero(t, l.total)
}

func TestCallerKey(t *testing.T) {
	t.Parallel()

	basic := func(userInfo string) string { return "Basic " + base64.StdEncoding.EncodeToString([]byte(userInfo)) }

	t.Run("SameCaller", func(t *testing.T) {
		t.Parallel()

		bearer := callerKey(callerAuth{authorization: "Bearer T"})
		assert.Equal(t, bearer, callerKey(callerAuth{authorization: basic("service_token:T")}))
		assert.Equal(t, bearer, callerKey(callerAuth{authorization: "Bearer T", cookie: "x=1"}))
		assert.Equal(t, bearer, callerKey(callerAuth{authorization: "Bearer T", cookie: "pmm_session=s"}))

		alice := callerKey(callerAuth{authorization: basic("alice:pw")})
		assert.Equal(t, alice, callerKey(callerAuth{authorization: basic("ALICE:pw")}))

		both := callerKey(callerAuth{authorization: basic("alice:pw"), cookie: "pmm_session=s"})
		assert.Equal(t, both, callerKey(callerAuth{authorization: basic("bob:x"), cookie: "pmm_session=t"}))
		assert.NotEqual(t, both, alice)
		assert.NotEqual(t, both, callerKey(callerAuth{cookie: "pmm_session=s"}))
		assert.NotEqual(t, both, callerKey(callerAuth{}))

		assert.Equal(t, callerKey(callerAuth{cookie: "pmm_session=abc"}), callerKey(callerAuth{cookie: "pmm_session=%61bc"}))

		session := callerKey(callerAuth{cookie: "pmm_session=s1"})
		assert.Equal(t, session, callerKey(callerAuth{cookie: "pmm_session=s1; other=1"}))
	})

	t.Run("DifferentCallers", func(t *testing.T) {
		t.Parallel()

		keys := []string{
			callerKey(callerAuth{authorization: "Bearer T1"}),
			callerKey(callerAuth{authorization: "Bearer T2"}),
			callerKey(callerAuth{authorization: basic("alice:pw")}),
			callerKey(callerAuth{authorization: basic("bob:pw")}),
			callerKey(callerAuth{cookie: "pmm_session=s1"}),
			callerKey(callerAuth{cookie: "pmm_session=s2"}),
			callerKey(callerAuth{}),
		}
		seen := map[string]bool{}
		for _, k := range keys {
			assert.False(t, seen[k], "duplicate key %s", k)
			seen[k] = true
		}
	})

	t.Run("IsSHA256", func(t *testing.T) {
		t.Parallel()

		sum := sha256.Sum256([]byte("token\x00T"))
		assert.Equal(t, hex.EncodeToString(sum[:]), callerKey(callerAuth{authorization: "Bearer T"}))
	})

	t.Run("NoSecretInKey", func(t *testing.T) {
		t.Parallel()

		for _, a := range []callerAuth{
			{authorization: "Bearer secrettoken"},
			{authorization: basic("aliceuser:pw")},
			{cookie: "pmm_session=sessionvalue"},
		} {
			key := callerKey(a)
			for _, secret := range []string{"secrettoken", "aliceuser", "sessionvalue"} {
				assert.NotContains(t, key, secret)
			}
		}
	})
}
