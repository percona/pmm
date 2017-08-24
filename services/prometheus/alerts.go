// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package prometheus

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

type AlertRule struct {
	Name     string
	FilePath string
	Text     string
	Disabled bool
}

// ListAlertRules returns all alert rules.
func (svc *Service) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	svc.lock.RLock()
	defer svc.lock.RUnlock()

	return svc.loadAlertRules(ctx)
}

// GetAlert return alert rule by name, or error if no such rule is present.
func (svc *Service) GetAlert(ctx context.Context, name string) (*AlertRule, error) {
	svc.lock.RLock()
	defer svc.lock.RUnlock()

	rules, err := svc.loadAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.Name == name {
			return &rule, nil
		}
	}
	return nil, errors.WithStack(os.ErrNotExist)
}

// PutAlert creates or replaces existing alert rule.
func (svc *Service) PutAlert(ctx context.Context, rule *AlertRule) error {
	// write to temporary location, check syntax with promtool
	f, err := ioutil.TempFile("", "pmm-managed-rule-")
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	if _, err = f.Write([]byte(rule.Text)); err != nil {
		return errors.WithStack(err)
	}
	b, err := exec.Command(svc.promtoolPath, "check-rules", f.Name()).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(b))
	}

	svc.lock.Lock()
	defer svc.lock.Unlock()

	// write to permanent location, reload configuration
	path := filepath.Join(svc.alertRulesPath, rule.Name)
	path += ".rule"
	if rule.Disabled {
		path += ".disabled"
	}
	if err := ioutil.WriteFile(path, []byte(rule.Text), 0666); err != nil {
		return errors.WithStack(err)
	}
	return svc.reload()
}

// DeleteAlert removes existing alert rule by name, or error if no such rule is present.
func (svc *Service) DeleteAlert(ctx context.Context, name string) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	rules, err := svc.loadAlertRules(ctx)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		if rule.Name == name {
			return os.Remove(rule.FilePath)
		}
	}
	return errors.WithStack(os.ErrNotExist)
}
