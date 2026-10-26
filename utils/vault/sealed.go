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
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// SealedPathPrefix is the path prefix of the references that carry a secret sealed to the key of a pmm-agent, so that
// it reaches Vault through that pmm-agent alone, without PMM Server, which passes it on, being able to read it:
// vault:pmm-agent-sealed/<sealed>#<key>, see Seal. They are not in Vault: only that pmm-agent opens them.
const SealedPathPrefix = "pmm-agent-sealed/"

// SealingAlgorithm names how a secret is sealed: ECDH on P-256 with an ephemeral key, HKDF with SHA-256 for an AES-256
// key, and AES-GCM. The sealed secret is the ephemeral public key, uncompressed, the nonce, and the ciphertext, in
// URL-safe base64 without padding.
const SealingAlgorithm = "ECDH-P256+HKDF-SHA256+AES-256-GCM"

// sealingInfo is the HKDF info that binds the derived key to its use.
const sealingInfo = "pmm-agent-sealed v1"

const (
	sealingPublicKeyLen = 65 // an uncompressed P-256 point
	sealingNonceLen     = 12
)

// IsSealedRef reports whether s is a reference that carries a sealed secret, see SealedPathPrefix.
func IsSealedRef(s string) bool {
	return strings.HasPrefix(s, RefPrefix+SealedPathPrefix)
}

// Seal seals value, the value of key in the Vault secret, to the public key of a pmm-agent, an uncompressed P-256
// point, and returns the reference that carries it.
func Seal(publicKey []byte, key, value string) (string, error) {
	recipient, err := ecdh.P256().NewPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("invalid sealing key: %w", err)
	}
	ephemeral, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	aead, err := sealingAEAD(ephemeral, recipient)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, sealingNonceLen)
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	sealed := append(ephemeral.PublicKey().Bytes(), nonce...)
	sealed = aead.Seal(sealed, nonce, []byte(value), nil)
	return RefPrefix + SealedPathPrefix + base64.RawURLEncoding.EncodeToString(sealed) + "#" + key, nil
}

// Open opens the secret sealed to privateKey that the path of a reference carries, see SealedPathPrefix.
func Open(privateKey *ecdh.PrivateKey, path string) (string, error) {
	encoded, ok := strings.CutPrefix(path, SealedPathPrefix)
	if !ok {
		return "", errors.New("not a sealed secret")
	}
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(sealed) < sealingPublicKeyLen+sealingNonceLen {
		return "", errors.New("malformed sealed secret")
	}

	ephemeral, err := ecdh.P256().NewPublicKey(sealed[:sealingPublicKeyLen])
	if err != nil {
		return "", fmt.Errorf("malformed sealed secret: %w", err)
	}
	aead, err := sealingAEAD(privateKey, ephemeral)
	if err != nil {
		return "", err
	}
	nonce := sealed[sealingPublicKeyLen : sealingPublicKeyLen+sealingNonceLen]
	value, err := aead.Open(nil, nonce, sealed[sealingPublicKeyLen+sealingNonceLen:], nil)
	if err != nil {
		return "", errors.New("the secret was sealed to another key: seal it again with the current key of the pmm-agent")
	}
	return string(value), nil
}

// sealingAEAD returns the AES-GCM cipher of the key that private and public agree on.
func sealingAEAD(private *ecdh.PrivateKey, public *ecdh.PublicKey) (cipher.AEAD, error) {
	shared, err := private.ECDH(public)
	if err != nil {
		return nil, err
	}
	key, err := hkdf.Key(sha256.New, shared, nil, sealingInfo, 32) //nolint:mnd // AES-256
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
