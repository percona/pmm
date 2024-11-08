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

// Package signatures verifies signatures received from Percona Platform.
package signatures

import (
	"github.com/percona/saas/pkg/check"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// defaultPublicKeys are the public keys used to download content from Percona Platform.
var defaultPublicKeys = []string{
	"RWTfyQTP3R7VzZggYY7dzuCbuCQWqTiGCqOvWRRAMVEiw0eSxHMVBBE5", // PMM 2.6
	"RWRxgu1w3alvJsQf+sHVUYiF6guAdEsBWXDe8jHZuB9dXVE9b5vw7ONM", // PMM 2.12
	"RWTHhufOlJ38dWt+DrprOg702YvZgqQJsx1XKfzF+MaB/pe9eCJgKkiF", // PMM 2.17
}

// Verify verifies checks signatures and returns error in case of verification problem.
func Verify(l *logrus.Entry, file string, signatures, publicKeys []string) error {
	if len(signatures) == 0 {
		return errors.New("zero signatures received")
	}

	if len(publicKeys) == 0 {
		publicKeys = defaultPublicKeys
	}

	var err error
	for _, sign := range signatures {
		for _, key := range publicKeys {
			if err = check.Verify([]byte(file), key, sign); err == nil {
				l.Debugf("Key %q matches signature %q.", key, sign)
				return nil
			}
			l.Debugf("Key %q doesn't match signature %q: %s.", key, sign, err)
		}
	}

	return errors.New("no verified signatures")
}
