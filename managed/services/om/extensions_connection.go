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

package om

import (
	"bytes"
	"cmp"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/env"
)

const (
	defaultExtensionsAddress    = "pmm-extensions:9000"
	defaultExtensionsSecretsDir = "/srv/extensions"
	// The side-car derives EXTENSIONS_INTERNAL_TOKEN as an HMAC of its SECRET_KEY under this label.
	internalTokenLabel = "extensions-internal-token"
)

// ResolveExtensions returns the PMM Extensions URL and bearer token. With neither PMM_EXTENSIONS_URL nor
// PMM_EXTENSIONS_TOKEN set, it targets the side-car PMM_ENABLE_EXTENSIONS provisions, as its nginx location does.
func ResolveExtensions(extensionsURL, token string, l *logrus.Entry) (string, string) {
	if extensionsURL != "" {
		if token != "" && cleartextToken(extensionsURL) {
			l.Warnf("PMM Extensions at %s is plain HTTP and off this host: its token will cross the network in clear text", extensionsURL)
		}
		return extensionsURL, token
	}
	if !env.ExtensionsEnabled() {
		return "", token
	}

	// No clear-text warning here: nginx already proxies browser sessions to this address over plain HTTP.
	extensionsURL = "http://" + cmp.Or(os.Getenv("PMM_EXTENSIONS_ADDRESS"), defaultExtensionsAddress)
	if token != "" {
		return extensionsURL, token
	}
	keyPath := filepath.Join(cmp.Or(os.Getenv("PMM_DEV_EXTENSIONS_SECRETS_DIR"), defaultExtensionsSecretsDir), "SECRET_KEY")
	token, err := deriveInternalToken(keyPath)
	if err != nil {
		l.Warnf("Cannot derive the PMM Extensions token, set PMM_EXTENSIONS_TOKEN: %s", err)
		return extensionsURL, ""
	}
	l.Infof("Using the PMM Extensions token derived from %s", keyPath)
	return extensionsURL, token
}

func deriveInternalToken(keyPath string) (string, error) {
	key, err := os.ReadFile(keyPath) //nolint:gosec
	if err != nil {
		return "", err
	}
	key = bytes.TrimSpace(key)
	if len(key) == 0 {
		return "", fmt.Errorf("%s is empty", keyPath)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(internalTokenLabel))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
