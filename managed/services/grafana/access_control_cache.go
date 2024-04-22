// Copyright (C) 2024 Percona LLC
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

package grafana

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const accessControlCacheExpiration = 3 * time.Second

// accessControl provides caching for the access control configuration.
type accessControl struct {
	mu sync.RWMutex

	db          *reform.DB
	enabled     bool
	lastUpdated time.Time
}

func (a *accessControl) isEnabled() bool {
	a.mu.RLock()

	if a.lastUpdated.Add(accessControlCacheExpiration).After(time.Now()) {
		defer a.mu.RUnlock()
		return a.enabled
	}

	a.mu.RUnlock()
	enabled, err := a.reload()
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return a.enabled
	}

	return enabled
}

func (a *accessControl) reload() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	settings, err := models.GetSettings(a.db.Querier)
	if err != nil {
		return false, err
	}

	a.enabled = settings.AccessControl.Enabled
	a.lastUpdated = time.Now()

	return a.enabled, nil
}
