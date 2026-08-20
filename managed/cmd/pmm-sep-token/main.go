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

// Package main generates the material for a Grafana service account token.
//
// It prints five lines on stdout, in the order the caller unpacks them: the
// token value, its api_key.key hash, and the user row's salt, rands and uid.
// The encoding reproduces Grafana's own - crc32("glsa_" + secret) little-endian
// as the checksum, util.EncodePassword(secret, checksum) as the stored hash.
//
// The secret is generated and hashed in-process so it never reaches any
// command line, which is also why this is a program rather than a shell
// pipeline over openssl.
package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"os"
	"strings"
)

const (
	tokenPrefix    = "glsa_"
	secretLength   = 32
	saltLength     = 10
	randsLength    = 10
	uidLength      = 14
	hashIterations = 10000
	hashLength     = 50

	byteValues = 256

	alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	lowerAlnum   = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// randomString returns n characters drawn uniformly from alphabet.
//
// Bytes at or above limit are discarded rather than folded in, because a plain
// modulo would over-represent the first byteValues%len(alphabet) characters.
func randomString(n int, alphabet string) (string, error) {
	limit := byteValues - byteValues%len(alphabet)
	buf := make([]byte, n)
	out := make([]byte, 0, n)

	for len(out) < n {
		_, err := rand.Read(buf)
		if err != nil {
			return "", fmt.Errorf("cannot read random bytes: %w", err)
		}
		for _, b := range buf {
			if int(b) >= limit {
				continue
			}
			out = append(out, alphabet[int(b)%len(alphabet)])
			if len(out) == n {
				break
			}
		}
	}

	return string(out), nil
}

// checksumFor reproduces Grafana's apikeygenprefix checksum: the CRC32 of the
// prefixed secret, little-endian, hex-encoded.
func checksumFor(secret string) string {
	sum := make([]byte, crc32.Size)
	binary.LittleEndian.PutUint32(sum, crc32.ChecksumIEEE([]byte(tokenPrefix+secret)))
	return hex.EncodeToString(sum)
}

// hashSecret reproduces Grafana's util.EncodePassword for a service account
// token: PBKDF2-HMAC-SHA256 over the secret, salted with the token's own
// checksum, hex-encoded.
func hashSecret(secret, checksum string) (string, error) {
	hashed, err := pbkdf2.Key(sha256.New, secret, []byte(checksum), hashIterations, hashLength)
	if err != nil {
		return "", fmt.Errorf("cannot derive the token hash: %w", err)
	}

	return hex.EncodeToString(hashed), nil
}

func generate() ([]string, error) {
	secret, err := randomString(secretLength, alphanumeric)
	if err != nil {
		return nil, err
	}
	salt, err := randomString(saltLength, alphanumeric)
	if err != nil {
		return nil, err
	}
	rands, err := randomString(randsLength, alphanumeric)
	if err != nil {
		return nil, err
	}
	uid, err := randomString(uidLength, lowerAlnum)
	if err != nil {
		return nil, err
	}

	checksum := checksumFor(secret)
	hashed, err := hashSecret(secret, checksum)
	if err != nil {
		return nil, err
	}

	return []string{
		tokenPrefix + secret + "_" + checksum,
		hashed,
		salt,
		rands,
		uid,
	}, nil
}

func main() {
	lines, err := generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pmm-sep-token: %s\n", err)
		os.Exit(1)
	}

	_, err = os.Stdout.WriteString(strings.Join(lines, "\n") + "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "pmm-sep-token: cannot write the token material: %s\n", err)
		os.Exit(1)
	}
}
