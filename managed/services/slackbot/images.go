// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package slackbot

import (
	"errors"
	"regexp"

	"github.com/percona/pmm/managed/services/grafana"
)

var (
	blobPathRE = regexp.MustCompile(`/v1/grafana/render/blob/([a-f0-9]{64})\.png`)
	hexHashRE  = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// ExtractBlobHashes finds Tier-1 render blob SHA-256 hashes referenced in analysis text (paths only).
func ExtractBlobHashes(analysis string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, m := range blobPathRE.FindAllStringSubmatch(analysis, -1) {
		if len(m) < 2 { //nolint:mnd
			continue
		}
		h := m[1]
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out
}

// LoadBlobPNG loads PNG bytes from the local Grafana render cache (trusted hash only).
func LoadBlobPNG(hash string) ([]byte, error) {
	if !hexHashRE.MatchString(hash) {
		return nil, errors.New("invalid blob hash")
	}
	b, err := grafana.ReadCachedRenderBlob(hash)
	if err != nil {
		return nil, err
	}
	return b, nil
}
