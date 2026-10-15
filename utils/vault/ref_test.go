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
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRef(t *testing.T) {
	t.Parallel()

	ref, err := ParseRef("vault:secret/data/pmm#admin_password")
	require.NoError(t, err)
	assert.Equal(t, Ref{Path: "secret/data/pmm", Key: "admin_password"}, ref)

	for in, expected := range map[string]string{
		"secret/data/pmm#key": "vault reference must start with vault:",
		"vault:secret/data":   "vault reference must have the form vault:<path>#<key>",
		"vault:secret/data#":  "vault reference must have the form vault:<path>#<key>",
		"vault:#key":          "vault path must not be empty",
		"vault:a/../b#key":    "vault path 'a/../b' has an empty or relative segment",
		"vault:a b#key":       "vault path 'a b' must not contain whitespace, '?' or '#'",
	} {
		_, err := ParseRef(in)
		require.EqualError(t, err, expected, in)
	}
}

func TestReplaceTokens(t *testing.T) {
	t.Parallel()

	secrets := map[string]map[string]string{
		"database/creds/pmm": {"username": "v-pmm", "password": "p@ss:w/rd {{ }}"},
	}
	get := func(path, key string) (string, error) {
		v, ok := secrets[path][key]
		if !ok {
			return "", errors.New("not found")
		}
		return v, nil
	}

	t.Run("NoTokens", func(t *testing.T) {
		t.Parallel()

		s, values, err := ReplaceTokens("user:pass@tcp(host)/", get)
		require.NoError(t, err)
		assert.Equal(t, "user:pass@tcp(host)/", s)
		assert.Empty(t, values)
	})

	t.Run("Raw", func(t *testing.T) {
		t.Parallel()

		in := Token("/database/creds/pmm/", "username", EncodingRaw) + ":" + Token("database/creds/pmm", "password", EncodingRaw) + "@tcp(host:3306)/"
		s, values, err := ReplaceTokens(in, get)
		require.NoError(t, err)
		assert.Equal(t, "v-pmm:p@ss:w/rd {{ }}@tcp(host:3306)/", s)
		assert.Equal(t, []string{"v-pmm", "p@ss:w/rd {{ }}"}, values)
	})

	t.Run("UserinfoSurvivesURLEncoding", func(t *testing.T) {
		t.Parallel()

		u := &url.URL{
			Scheme: "postgres",
			Host:   "host:5432",
			User: url.UserPassword(
				Token("database/creds/pmm", "username", EncodingUserinfo),
				Token("database/creds/pmm", "password", EncodingUserinfo),
			),
		}
		s, values, err := ReplaceTokens(u.String(), get)
		require.NoError(t, err)

		parsed, err := url.Parse(s)
		require.NoError(t, err)
		password, _ := parsed.User.Password()
		assert.Equal(t, "v-pmm", parsed.User.Username())
		assert.Equal(t, "p@ss:w/rd {{ }}", password)
		assert.Contains(t, values, "p@ss:w/rd {{ }}")
		assert.Contains(t, values, "p%40ss%3Aw%2Frd%20%7B%7B%20%7D%7D")
	})

	t.Run("QuerySurvivesURLEncoding", func(t *testing.T) {
		t.Parallel()

		q := url.Values{"tlsCertificateKeyFilePassword": []string{Token("database/creds/pmm", "password", EncodingQuery)}}
		s, values, err := ReplaceTokens("mongodb://host:27017/?"+q.Encode(), get)
		require.NoError(t, err)

		parsed, err := url.Parse(s)
		require.NoError(t, err)
		assert.Equal(t, "p@ss:w/rd {{ }}", parsed.Query().Get("tlsCertificateKeyFilePassword"))
		assert.Contains(t, values, "p%40ss%3Aw%2Frd+%7B%7B+%7D%7D")
	})

	t.Run("MissingKey", func(t *testing.T) {
		t.Parallel()

		_, _, err := ReplaceTokens(Token("database/creds/pmm", "nope", EncodingRaw), get)
		require.EqualError(t, err, "not found")
	})

	t.Run("UnknownEncoding", func(t *testing.T) {
		t.Parallel()

		_, _, err := ReplaceTokens(Token("database/creds/pmm", "username", "base32"), get)
		require.EqualError(t, err, "unsupported vault token encoding 'base32'")
	})
}

func TestTokenWithFallback(t *testing.T) {
	t.Parallel()

	get := func(path, key string) (string, error) {
		switch key {
		case "username":
			return "", &KeyNotFoundError{Path: path, Key: key}
		case "password":
			return "s3cr3t", nil
		default:
			return "", errors.New("vault is down")
		}
	}

	s, values, err := ReplaceTokens(TokenWithFallback("kv/db", "username", EncodingUserinfo, "pmm user")+":"+Token("kv/db", "password", EncodingRaw), get)
	require.NoError(t, err)
	assert.Equal(t, "pmm%20user:s3cr3t", s)
	assert.Equal(t, []string{"s3cr3t"}, values, "a value with a fallback is not a secret to redact")

	// Only a missing key falls back, not a failed read.
	_, _, err = ReplaceTokens(TokenWithFallback("kv/db", "other", EncodingRaw, "x"), get)
	require.EqualError(t, err, "vault is down")

	// A secret that has the key wins over the fallback.
	s, _, err = ReplaceTokens(TokenWithFallback("kv/db", "password", EncodingRaw, "fallback"), get)
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", s)
}

func TestTokenForRef(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Token("secret/data/pmm/mysql", "tls_key", EncodingRaw), TokenForRef("vault:secret/data/pmm/mysql#tls_key", EncodingRaw))
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----", TokenForRef("-----BEGIN PRIVATE KEY-----", EncodingRaw))
	assert.Equal(t, "vault:no-key", TokenForRef("vault:no-key", EncodingRaw))
}
