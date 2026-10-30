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

package tlshelpers

import (
	"context"

	"github.com/percona/pmm/agent/utils/secrets"
)

// resolveFiles returns the contents of the files with the Vault tokens in them replaced with the secrets
// they stand for: a client key or certificate may be stored in Vault. The given map is not changed.
func resolveFiles(ctx context.Context, files map[string]string) (map[string]string, error) {
	res := make(map[string]string, len(files))
	for name, content := range files {
		v, err := secrets.ResolveString(ctx, content)
		if err != nil {
			return nil, err
		}
		res[name] = v
	}
	return res, nil
}
