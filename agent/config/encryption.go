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

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/youmark/pkcs8"
)

// Encryption handles encryption and decryption of data using hybrid RSA + AES-GCM scheme.
type Encryption struct {
	KeyFile         string
	KeyFilePassword string
}

const (
	gcmNonceSize = 12
	aesKeySize   = 32
)

// Encrypt encrypts the given plaintext.
func (enc Encryption) Encrypt(plain []byte) ([]byte, error) {
	priv, err := enc.readKeyFile()
	if err != nil {
		return nil, fmt.Errorf("unable to get RSA key from KeyFile: %w", err)
	}

	aesKey := make([]byte, aesKeySize)
	_, err = io.ReadFull(rand.Reader, aesKey)
	if err != nil {
		return nil, fmt.Errorf("unable to generate AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("unable to init AES: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		return nil, fmt.Errorf("unable to init GCM: %w", err)
	}
	nonce := make([]byte, gcmNonceSize)
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, fmt.Errorf("unable to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)

	wrappedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &priv.PublicKey, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to RSA-wrap AES key: %w", err)
	}

	out := make([]byte, 0, len(wrappedKey)+len(nonce)+len(ciphertext))
	out = append(out, wrappedKey...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt decrypts the given ciphertext.
func (enc Encryption) Decrypt(in []byte) ([]byte, error) {
	priv, err := enc.readKeyFile()
	if err != nil {
		return nil, fmt.Errorf("unable to get RSA key from KeyFile: %w", err)
	}

	k := priv.Size()
	if len(in) < k+gcmNonceSize+1 {
		return nil, errors.New("ciphertext too short")
	}

	wrappedKey := in[:k]
	nonce := in[k : k+gcmNonceSize]
	ciphertext := in[k+gcmNonceSize:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, wrappedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to RSA-unwrap AES key: %w", err)
	}
	if len(aesKey) != aesKeySize {
		return nil, fmt.Errorf("unexpected AES key length: %d", len(aesKey))
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("unable to init AES: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		return nil, fmt.Errorf("unable to init GCM: %w", err)
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt (wrong key or data tampered): %w", err)
	}
	return plain, nil
}

func (enc Encryption) readKeyFile() (*rsa.PrivateKey, error) {
	f, err := os.ReadFile(enc.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read KeyFile: %w", err)
	}

	block, _ := pem.Decode(f)
	if block == nil {
		return nil, errors.New("no valid private key found in a KeyFile")
	}

	k, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(enc.KeyFilePassword))
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}
