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

// Package bcrypt provides a our bcrypt implementation changes compared to upstream.
package bcrypt

import (
	"encoding/base64"
	"fmt"

	upstream "golang.org/x/crypto/bcrypt"
	//nolint:staticcheck // SA1019: using blowfish intentionally for bcrypt
	"golang.org/x/crypto/blowfish"
)

const (
	// MaxSaltSize is the maximum salt length bcrypt supports. Exported for callers.
	// Must be aligned with maxSaltSize in upstream package (bcrypt/bcrypt.go).
	MaxSaltSize = 16

	alphabet = "./ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	majorVersion = '2'
	minorVersion = 'a'

	// MinCost is the minimum allowable cost as passed in to GenerateFromPassword.
	MinCost int = 4
	// MaxCost is the maximum allowable cost as passed in to GenerateFromPassword.
	MaxCost int = 31
	// DefaultCost is the cost that will actually be set if a cost below MinCost is passed into GenerateFromPassword.
	DefaultCost int = 10

	maxCryptedHashSize   = 23
	encodedSaltSize      = 22
	encodedHashSize      = 31
	encodedHashArraySize = 60
	base64Pad            = 4
)

var bcEncoding = base64.NewEncoding(alphabet)

// magicCipherData is an IV for the 64 Blowfish encryption calls in
// bcrypt(). It's the string "OrpheanBeholderScryDoubt" in big-endian bytes.
//
//nolint:go-consistent
var magicCipherData = []byte{
	0x4f, 0x72, 0x70, 0x68,
	0x65, 0x61, 0x6e, 0x42,
	0x65, 0x68, 0x6f, 0x6c,
	0x64, 0x65, 0x72, 0x53,
	0x63, 0x72, 0x79, 0x44,
	0x6f, 0x75, 0x62, 0x74,
}

type hashed struct {
	hash  []byte
	salt  []byte
	cost  int // allowed range is MinCost to MaxCost
	major byte
	minor byte
}

func (p *hashed) Hash() []byte {
	arr := make([]byte, encodedHashArraySize)
	arr[0] = '$'
	arr[1] = p.major
	n := 2

	if p.minor != 0 {
		arr[2] = p.minor
		n = 3
	}
	arr[n] = '$'
	n++
	costStr := fmt.Sprintf("%02d", p.cost)
	copy(arr[n:], costStr)
	n += len(costStr)
	arr[n] = '$'
	n++
	copy(arr[n:], p.salt)
	n += encodedSaltSize
	copy(arr[n:], p.hash)
	n += encodedHashSize
	return arr[:n]
}

// GenerateFromPasswordAndSalt returns the bcrypt hash of the password at the given
// cost using custom salt. If the cost given is less than MinCost, the cost will
// be set to DefaultCost, instead.
func GenerateFromPasswordAndSalt(password []byte, cost int, salt []byte) ([]byte, error) {
	p, err := newFromPasswordAndSalt(password, cost, salt)
	if err != nil {
		return nil, err
	}
	return p.Hash(), nil
}

func newFromPasswordAndSalt(password []byte, cost int, salt []byte) (*hashed, error) {
	if cost < upstream.MinCost {
		cost = upstream.DefaultCost
	}
	p := new(hashed) //nolint:go-consistent
	p.major = majorVersion
	p.minor = minorVersion

	err := checkCost(cost)
	if err != nil {
		return nil, err
	}
	p.cost = cost

	p.salt = base64Encode(salt)
	hash, err := bcrypt(password, p.cost, p.salt)
	if err != nil {
		return nil, err
	}
	p.hash = hash
	return p, err
}

func checkCost(cost int) error {
	if cost < MinCost || cost > MaxCost {
		return upstream.InvalidCostError(cost)
	}
	return nil
}

func bcrypt(password []byte, cost int, salt []byte) ([]byte, error) {
	cipherData := make([]byte, len(magicCipherData))
	copy(cipherData, magicCipherData)

	c, err := expensiveBlowfishSetup(password, cost, salt)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 24; i += 8 {
		for range 64 {
			c.Encrypt(cipherData[i:i+8], cipherData[i:i+8])
		}
	}

	// Bug compatibility with C bcrypt implementations. We only encode 23 of
	// the 24 bytes encrypted.
	hsh := base64Encode(cipherData[:maxCryptedHashSize])
	return hsh, nil
}

func expensiveBlowfishSetup(key []byte, cost int, salt []byte) (*blowfish.Cipher, error) {
	csalt, err := base64Decode(salt)
	if err != nil {
		return nil, err
	}

	// Bug compatibility with C bcrypt implementations. They use the trailing
	// NULL in the key string during expansion.
	// We copy the key to prevent changing the underlying array.
	ckey := make([]byte, len(key)+1)
	copy(ckey, key)
	ckey[len(key)] = 0

	c, err := blowfish.NewSaltedCipher(ckey, csalt)
	if err != nil {
		return nil, err
	}

	rounds := uint64(1) << uint64(cost) //nolint:gosec
	for range rounds {
		blowfish.ExpandKey(ckey, c)
		blowfish.ExpandKey(csalt, c)
	}

	return c, nil
}

func base64Encode(src []byte) []byte {
	n := bcEncoding.EncodedLen(len(src))
	dst := make([]byte, n)
	bcEncoding.Encode(dst, src)

	for n > 0 && dst[n-1] == '=' {
		n--
	}
	return dst[:n]
}

func base64Decode(src []byte) ([]byte, error) {
	numOfEquals := (base64Pad - (len(src) % base64Pad)) % base64Pad
	if numOfEquals > 0 {
		for range numOfEquals {
			src = append(src, '=')
		}
	}

	dst := make([]byte, bcEncoding.DecodedLen(len(src)))
	n, err := bcEncoding.Decode(dst, src)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}
