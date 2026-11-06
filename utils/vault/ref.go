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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// RefPrefix starts a reference to a value stored in Vault: vault:<path>#<key>.
const RefPrefix = "vault:"

// Ref points to the value of a key in a Vault secret.
type Ref struct {
	Path string
	Key  string
}

// IsRef reports whether s is a reference to a value stored in Vault.
func IsRef(s string) bool {
	return strings.HasPrefix(s, RefPrefix)
}

// StagedPathPrefix is the path prefix of the references to the secrets that pmm-agent holds until PMM Server asks it
// to store them in Vault. The secrets to store are staged there by pmm-admin, so that they reach Vault through
// pmm-agent alone, and PMM Server gets these references instead. They are not in Vault: only that pmm-agent
// resolves them.
const StagedPathPrefix = "pmm-agent-staged/"

// IsStagedRef reports whether s is a reference to a secret staged in pmm-agent, see StagedPathPrefix.
func IsStagedRef(s string) bool {
	return strings.HasPrefix(s, RefPrefix+StagedPathPrefix)
}

// ToStore reports whether s is a secret to store in Vault: one given as is, or staged in or sealed to pmm-agent,
// rather than a reference to one that is in Vault already.
func ToStore(s string) bool {
	return s != "" && (!IsRef(s) || IsStagedRef(s) || IsSealedRef(s))
}

// ParseRef parses a vault:<path>#<key> reference.
func ParseRef(s string) (Ref, error) {
	rest, ok := strings.CutPrefix(s, RefPrefix)
	if !ok {
		return Ref{}, fmt.Errorf("vault reference must start with %s", RefPrefix)
	}

	i := strings.LastIndex(rest, "#")
	if i < 0 || rest[i+1:] == "" {
		return Ref{}, errors.New("vault reference must have the form vault:<path>#<key>")
	}

	ref := Ref{Path: rest[:i], Key: rest[i+1:]}
	err := ValidatePath(ref.Path)
	if err != nil {
		return Ref{}, err
	}
	return ref, nil
}

// ValidatePath returns an error if path cannot be the API path of a Vault secret.
func ValidatePath(path string) error {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return errors.New("vault path must not be empty")
	}
	if strings.ContainsAny(trimmed, " \t\r\n?#") {
		return fmt.Errorf("vault path '%s' must not contain whitespace, '?' or '#'", path)
	}
	for s := range strings.SplitSeq(trimmed, "/") {
		if s == "" || s == "." || s == ".." {
			return fmt.Errorf("vault path '%s' has an empty or relative segment", path)
		}
	}
	return nil
}

// Encoding is how a secret value is escaped where a token stands for it.
type Encoding string

const (
	// EncodingRaw inserts the value as is.
	EncodingRaw Encoding = "raw"
	// EncodingUserinfo escapes the value for the userinfo of a URL.
	EncodingUserinfo Encoding = "userinfo"
	// EncodingQuery escapes the value for the query of a URL.
	EncodingQuery Encoding = "query"
)

const tokenStart = "~pmm-vault~" //nolint:gosec

var tokenRE = regexp.MustCompile(`~pmm-vault~([A-Za-z0-9_-]+)~`) //nolint:gochecknoglobals

type tokenPayload struct {
	Path     string   `json:"p"`
	Key      string   `json:"k"`
	Encoding Encoding `json:"e"`
	Fallback *string  `json:"f,omitempty"`
}

// Token returns a placeholder that pmm-agent replaces with the value of key in the secret at path,
// escaped as enc. pmm-managed puts tokens where database credentials go, so that the credentials
// themselves never leave Vault for PMM Server. A token consists of URL-unreserved characters only:
// it passes unchanged through the URL, DSN and template encoding of the strings it is put in.
func Token(path, key string, enc Encoding) string {
	return encodeToken(tokenPayload{Path: strings.Trim(path, "/"), Key: key, Encoding: enc})
}

// TokenWithFallback is Token for a key the secret may not have: fallback stands in for its value then.
func TokenWithFallback(path, key string, enc Encoding, fallback string) string {
	return encodeToken(tokenPayload{Path: strings.Trim(path, "/"), Key: key, Encoding: enc, Fallback: &fallback})
}

func encodeToken(p tokenPayload) string {
	b, _ := json.Marshal(p) //nolint:errchkjson
	return tokenStart + base64.RawURLEncoding.EncodeToString(b) + "~"
}

// TokenForRef returns the token for the value s refers to if s is a vault:<path>#<key> reference, and s
// itself otherwise, such as for a secret stored in PMM Server or an invalid reference, which validation
// refuses before it is stored.
func TokenForRef(s string, enc Encoding) string {
	ref, err := ParseRef(s)
	if err != nil {
		return s
	}
	return Token(ref.Path, ref.Key, enc)
}

// HasTokens reports whether s contains a token.
func HasTokens(s string) bool {
	return strings.Contains(s, tokenStart)
}

// Getter returns the value of key in the secret at path.
type Getter func(path, key string) (string, error)

// ReplaceTokens replaces the tokens in s with the values get returns. Alongside the result it returns
// the secrets it inserted, both as read and as escaped, for redacting them from logs. A value with a
// fallback, such as a username, is not one: its fallback is in the token in plain text.
func ReplaceTokens(s string, get Getter) (string, []string, error) {
	if !HasTokens(s) {
		return s, nil, nil
	}

	var values []string
	var firstErr error
	res := tokenRE.ReplaceAllStringFunc(s, func(token string) string {
		if firstErr != nil {
			return token
		}

		v, escaped, secret, err := resolveToken(token, get)
		if err != nil {
			firstErr = err
			return token
		}

		if secret {
			values = append(values, v)
			if escaped != v {
				values = append(values, escaped)
			}
		}
		return escaped
	})
	if firstErr != nil {
		return "", nil, firstErr
	}

	return res, values, nil
}

// resolveToken returns the value a token stands for, that value escaped, and whether it is a secret.
func resolveToken(token string, get Getter) (string, string, bool, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(token, tokenStart), "~"))
	if err != nil {
		return "", "", false, fmt.Errorf("invalid vault token: %w", err)
	}

	var p tokenPayload
	err = json.Unmarshal(b, &p)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid vault token: %w", err)
	}

	v, err := get(p.Path, p.Key)
	_, missing := errors.AsType[*KeyNotFoundError](err)
	if missing && p.Fallback != nil {
		v, err = *p.Fallback, nil
	}
	if err != nil {
		return "", "", false, err
	}

	secret := p.Fallback == nil
	switch p.Encoding {
	case EncodingRaw:
		return v, v, secret, nil
	case EncodingUserinfo:
		return v, url.User(v).String(), secret, nil
	case EncodingQuery:
		return v, url.QueryEscape(v), secret, nil
	default:
		return "", "", false, fmt.Errorf("unsupported vault token encoding '%s'", p.Encoding)
	}
}
