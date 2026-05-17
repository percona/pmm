// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package querylog

import (
	"strconv"
	"strings"

	"github.com/go-faster/city"
)

// fingerprint normalizes a ClickHouse SQL statement into a stable query class
// representation by stripping literals. It is a lexer-based pass — not a full
// SQL parser — so it degrades gracefully: on any input it produces a usable
// (if coarse) fingerprint rather than failing. Behaviour:
//
//   - line (`-- …`) and block (`/* … */`) comments are removed;
//   - string literals ('…' incl. ” / \' escapes) collapse to `?`;
//   - numeric literals (incl. hex 0x…, floats, exponents) collapse to `?`;
//   - a run of `?` separated only by commas/whitespace collapses to a single
//     `?`, so `IN (?, ?, ?)`, arrays `[?, ?]` and tuples `(?, ?)` normalize;
//   - `LIMIT n` / `LIMIT n, m` / `LIMIT n OFFSET m` collapse to `LIMIT ?`;
//   - `{name:Type}` server-side query parameters are preserved verbatim;
//   - runs of whitespace collapse to a single space.
func fingerprint(query string) string {
	stripped := stripComments(query)

	var b strings.Builder
	b.Grow(len(stripped))
	runes := []rune(stripped)

	for i := 0; i < len(runes); {
		c := runes[i]
		switch {
		case c == '\'':
			// String literal — scan to the closing quote, honouring '' and \' escapes.
			i++
			for i < len(runes) {
				if runes[i] == '\\' && i+1 < len(runes) {
					i += 2
					continue
				}
				if runes[i] == '\'' {
					if i+1 < len(runes) && runes[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			b.WriteByte('?')

		case c == '{':
			// Preserve {name:Type} query parameter placeholders verbatim.
			j := i
			for j < len(runes) && runes[j] != '}' {
				j++
			}
			if j < len(runes) {
				b.WriteString(string(runes[i : j+1]))
				i = j + 1
			} else {
				b.WriteRune(c)
				i++
			}

		case isDigitStart(runes, i):
			// Numeric literal — only when it is not part of an identifier.
			i = skipNumber(runes, i)
			b.WriteByte('?')

		default:
			b.WriteRune(c)
			i++
		}
	}

	out := collapsePlaceholders(b.String())
	out = normalizeLimit(out)
	out = strings.Join(strings.Fields(out), " ")
	return strings.TrimSpace(out)
}

// isDigitStart reports whether position i begins a numeric literal rather than
// sitting inside an identifier (e.g. the "1" in column "col1" is not a literal).
func isDigitStart(runes []rune, i int) bool {
	c := runes[i]
	if c < '0' || c > '9' {
		return false
	}
	if i > 0 {
		p := runes[i-1]
		if p == '_' || p == '$' ||
			(p >= 'a' && p <= 'z') || (p >= 'A' && p <= 'Z') ||
			(p >= '0' && p <= '9') {
			return false
		}
	}
	return true
}

// skipNumber advances past a numeric literal starting at i: decimal, float,
// scientific notation, or 0x… hexadecimal.
func skipNumber(runes []rune, i int) int {
	n := len(runes)
	if i+1 < n && runes[i] == '0' && (runes[i+1] == 'x' || runes[i+1] == 'X') {
		i += 2
		for i < n && isHexDigit(runes[i]) {
			i++
		}
		return i
	}
	for i < n {
		c := runes[i]
		switch {
		case c >= '0' && c <= '9', c == '.':
			i++
		case (c == 'e' || c == 'E') && i+1 < n &&
			(runes[i+1] == '+' || runes[i+1] == '-' || (runes[i+1] >= '0' && runes[i+1] <= '9')):
			i += 2
		default:
			return i
		}
	}
	return i
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// stripComments removes -- line comments and /* */ block comments while leaving
// such sequences inside string literals untouched.
func stripComments(query string) string {
	var b strings.Builder
	b.Grow(len(query))
	runes := []rune(query)

	for i := 0; i < len(runes); {
		c := runes[i]
		switch {
		case c == '\'':
			b.WriteRune(c)
			i++
			for i < len(runes) {
				b.WriteRune(runes[i])
				if runes[i] == '\\' && i+1 < len(runes) {
					b.WriteRune(runes[i+1])
					i += 2
					continue
				}
				if runes[i] == '\'' {
					i++
					break
				}
				i++
			}
		case c == '-' && i+1 < len(runes) && runes[i+1] == '-':
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(runes) && runes[i+1] == '*':
			i += 2
			for i+1 < len(runes) && (runes[i] != '*' || runes[i+1] != '/') {
				i++
			}
			i += 2
			b.WriteByte(' ')
		default:
			b.WriteRune(c)
			i++
		}
	}
	return b.String()
}

// collapsePlaceholders reduces any run of `?` separated solely by commas and
// whitespace to a single `?`, normalizing IN lists, arrays and tuples.
func collapsePlaceholders(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)

	for i := 0; i < len(runes); {
		if runes[i] == '?' {
			j := i + 1
			sawSpace := false
			for j < len(runes) {
				c := runes[j]
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					sawSpace = true
					j++
					continue
				}
				if c == ',' {
					j++
					continue
				}
				if c == '?' {
					j++
					sawSpace = false
					continue
				}
				break
			}
			b.WriteByte('?')
			// A run that consumed whitespace before a non-`?` token must keep
			// one space so adjacent tokens are not glued together.
			if sawSpace && j < len(runes) && runes[j] != ')' && runes[j] != ']' && runes[j] != ',' {
				b.WriteByte(' ')
			}
			i = j
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

// normalizeLimit rewrites `LIMIT ?[, ?]` and `LIMIT ? OFFSET ?` to `LIMIT ?`.
// Numeric arguments are already `?` at this stage (collapsed by the literal
// pass), so only the surrounding `, ?` / `OFFSET ?` tail needs trimming.
func normalizeLimit(s string) string {
	lower := strings.ToLower(s)
	idx := strings.Index(lower, "limit ?")
	if idx < 0 {
		return s
	}
	head := s[:idx+len("limit ?")]
	rest := strings.TrimLeft(s[idx+len("limit ?"):], " \t")
	switch {
	case strings.HasPrefix(rest, ","):
		rest = strings.TrimLeft(rest[1:], " \t")
		rest = strings.TrimPrefix(rest, "?")
	case strings.HasPrefix(strings.ToLower(rest), "offset ?"):
		rest = rest[len("offset ?"):]
	default:
		return s
	}
	return head + rest
}

// hashFingerprint returns the hex-encoded query class identifier. When a
// non-zero ClickHouse normalized_query_hash is supplied it is used directly so
// that the agent's grouping is identical to the server's own; otherwise the
// normalized fingerprint is hashed client-side.
//
// Client-side hashing uses go-faster/city's CH64 — the cityHash64 variant
// ClickHouse itself uses for normalized_query_hash. The package is already a
// transitive dependency (via ClickHouse/ch-go), so no new module is added and
// server-side and client-side hashes are produced by the same algorithm.
func hashFingerprint(serverHash uint64, fingerprint string) string {
	if serverHash != 0 {
		return strconv.FormatUint(serverHash, 16)
	}
	return strconv.FormatUint(city.CH64([]byte(fingerprint)), 16)
}
