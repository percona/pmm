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

package secrets

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/percona/pmm/utils/vault"
)

const (
	// How long a staged secret is held: pmm-admin stages it right before it sends PMM Server the request that has
	// it stored.
	stagedTTL = 10 * time.Minute

	// The most secrets staged at once, which anyone who reaches the local API can stage.
	maxStaged = 100
)

type stagedSecret struct {
	data    map[string]string
	expires time.Time
}

var staged = struct { //nolint:gochecknoglobals
	m       sync.Mutex
	secrets map[string]stagedSecret
	now     func() time.Time
}{
	secrets: make(map[string]stagedSecret),
	now:     time.Now,
}

// ErrStagedNotFound is returned for a reference to a staged secret that pmm-agent does not hold: it was stored
// already, it expired, or another pmm-agent staged it.
var ErrStagedNotFound = errors.New("the secret staged in pmm-agent was not found, it may have expired")

// Stage holds data until PMM Server asks pmm-agent to store it in Vault, and returns a reference to each of its
// keys, see vault.StagedPathPrefix. The references are resolved by this pmm-agent alone, and only for a while.
func Stage(data map[string]string) (map[string]string, error) {
	b := make([]byte, 16) //nolint:mnd
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	handle := hex.EncodeToString(b)

	staged.m.Lock()
	defer staged.m.Unlock()

	now := staged.now()
	for h, s := range staged.secrets {
		if !now.Before(s.expires) {
			delete(staged.secrets, h)
		}
	}
	if len(staged.secrets) >= maxStaged {
		return nil, errors.New("too many secrets are staged in pmm-agent, try again later")
	}
	staged.secrets[handle] = stagedSecret{data: maps.Clone(data), expires: now.Add(stagedTTL)}

	refs := make(map[string]string, len(data))
	for key := range data {
		refs[key] = vault.RefPrefix + vault.StagedPathPrefix + handle + "#" + key
	}
	return refs, nil
}

// stagedValue returns the value of key in the staged secret that path names, and reports whether path names
// one at all.
func stagedValue(path, key string) (string, bool, error) {
	handle, ok := strings.CutPrefix(path, vault.StagedPathPrefix)
	if !ok {
		return "", false, nil
	}

	staged.m.Lock()
	defer staged.m.Unlock()

	s, found := staged.secrets[handle]
	if !found || !staged.now().Before(s.expires) {
		return "", true, ErrStagedNotFound
	}
	v, found := s.data[key]
	if !found {
		return "", true, ErrStagedNotFound
	}
	return v, true, nil
}

// unstage returns a copy of data with the references to staged secrets replaced with the secrets, and the
// handles of the staged secrets.
func unstage(data map[string]string) (map[string]string, []string, error) {
	res := make(map[string]string, len(data))
	var handles []string
	for key, value := range data {
		if !vault.IsStagedRef(value) {
			res[key] = value
			continue
		}

		ref, err := vault.ParseRef(value)
		if err != nil {
			return nil, nil, err
		}
		v, _, err := stagedValue(ref.Path, ref.Key)
		if err != nil {
			return nil, nil, err
		}
		res[key] = v
		handles = append(handles, strings.TrimPrefix(ref.Path, vault.StagedPathPrefix))
	}
	return res, handles, nil
}

// dropStaged drops the staged secrets with the given handles, once stored in Vault.
func dropStaged(handles []string) {
	staged.m.Lock()
	defer staged.m.Unlock()

	for _, h := range handles {
		delete(staged.secrets, h)
	}
}
