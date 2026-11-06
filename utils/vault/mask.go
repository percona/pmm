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
	"regexp"
	"strings"
)

// configSecretLineRE matches the lines of a YAML configuration that may hold a credential: the passwords,
// including those of basic authentication, the tokens, and the Vault secret ID.
var configSecretLineRE = regexp.MustCompile(`(?m)^([ \t]*(?:password|token|bearer_token|secret-id):[ \t]*)(\S.*)$`) //nolint:gochecknoglobals

// MaskConfigSecrets returns a YAML configuration with the values of the credentials it holds masked, for
// including it in a support bundle. A vault:<path>#<key> reference is not a credential, so it is kept.
func MaskConfigSecrets(b []byte) []byte {
	return configSecretLineRE.ReplaceAllFunc(b, func(line []byte) []byte {
		m := configSecretLineRE.FindSubmatch(line)
		if IsRef(strings.Trim(string(m[2]), `"'`)) {
			return line
		}
		return append(m[1], "'***'"...)
	})
}
