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
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/youmark/pkcs8"
)

type Encryption struct {
	KeyFile         string
	KeyFilePassword string
}

const gcmNonceSize = 12

func (enc Encryption) Encrypt(plain []byte) ([]byte, error) {
	priv, err := enc.readKeyFile()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get RSA key from KeyFile")
	}

	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, errors.Wrap(err, "unable to generate AES key")
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init AES")
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init GCM")
	}
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Wrap(err, "unable to generate nonce")
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)

	wrappedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &priv.PublicKey, aesKey, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to RSA-wrap AES key")
	}

	out := make([]byte, 0, len(wrappedKey)+len(nonce)+len(ciphertext))
	out = append(out, wrappedKey...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (enc Encryption) Decrypt(in []byte) ([]byte, error) {
	priv, err := enc.readKeyFile()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get RSA key from KeyFile")
	}

	k := priv.PublicKey.Size()
	if len(in) < k+gcmNonceSize+1 {
		return nil, errors.New("ciphertext too short")
	}

	wrappedKey := in[:k]
	nonce := in[k : k+gcmNonceSize]
	ciphertext := in[k+gcmNonceSize:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, wrappedKey, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to RSA-unwrap AES key")
	}
	if len(aesKey) != 32 {
		return nil, errors.Errorf("unexpected AES key length: %d", len(aesKey))
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init AES")
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init GCM")
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decrypt (wrong key or data tampered)")
	}
	return plain, nil
}

func (enc Encryption) readKeyFile() (*rsa.PrivateKey, error) {
	f, err := os.ReadFile(enc.KeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read KeyFile")
	}

	block, _ := pem.Decode(f)
	if block == nil {
		return nil, errors.New("no valid private key found in a KeyFile")
	}

	k, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(enc.KeyFilePassword))
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse private key")
	}

	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}
