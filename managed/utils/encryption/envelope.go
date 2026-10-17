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

package encryption

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// EnvelopePrefix marks values stored in the self-describing envelope format:
// the prefix followed by base64 of a Tink ciphertext. Tink's TINK output
// prefix embeds the encrypting key ID in the ciphertext, so an envelope value
// carries both its encryption state and its key without external bookkeeping.
const EnvelopePrefix = "pmm1$"

const (
	// Tink's TINK output prefix: one format byte followed by a big-endian
	// uint32 key ID.
	tinkPrefixLen  = 5
	tinkPrefixByte = 0x01
)

// ErrMalformedEnvelope is reported by Inspect callers for a value that starts
// with EnvelopePrefix but is not base64 of a Tink ciphertext. Every envelope
// PMM writes is well-formed, so such a value is plaintext, or a corrupted
// envelope where only ciphertext can be stored.
var ErrMalformedEnvelope = errors.New("value starts with the envelope prefix but is not an envelope")

// IsEncrypted reports whether a stored value is in the envelope format: the
// prefix followed by base64 of a Tink ciphertext. A value that only starts
// with the prefix is not; see ErrMalformedEnvelope.
func IsEncrypted(stored string) bool {
	_, err := unwrapEnvelope(stored)
	return err == nil
}

// HasEnvelopePrefix reports whether a stored value starts with EnvelopePrefix,
// whether or not it is a well-formed envelope.
func HasEnvelopePrefix(stored string) bool {
	return strings.HasPrefix(stored, EnvelopePrefix)
}

func wrapEnvelope(ciphertext []byte) string {
	return EnvelopePrefix + base64.StdEncoding.EncodeToString(ciphertext)
}

func unwrapEnvelope(stored string) ([]byte, error) {
	encoded, ok := strings.CutPrefix(stored, EnvelopePrefix)
	if !ok {
		return nil, fmt.Errorf("value is not in the %q envelope format", EnvelopePrefix)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedEnvelope, err)
	}
	if _, ok := tinkKeyID(ciphertext); !ok || len(ciphertext) < minLegacyCiphertextLen {
		return nil, fmt.Errorf("%w: not a Tink ciphertext", ErrMalformedEnvelope)
	}

	return ciphertext, nil
}

func tinkKeyID(ciphertext []byte) (uint32, bool) {
	if len(ciphertext) <= tinkPrefixLen || ciphertext[0] != tinkPrefixByte {
		return 0, false
	}

	return binary.BigEndian.Uint32(ciphertext[1:tinkPrefixLen]), true
}

// StoredKeyID returns the ID of the key that encrypted an envelope-format
// value; ok is false for plaintext and legacy values.
func StoredKeyID(stored string) (uint32, bool) {
	ciphertext, err := unwrapEnvelope(stored)
	if err != nil {
		return 0, false
	}

	return tinkKeyID(ciphertext)
}
