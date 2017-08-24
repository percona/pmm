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

package services

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
func (p *Prometheus) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.loadAlertRules(ctx)
}

// GetAlert return alert rule by name, or error if no such rule is present.
func (p *Prometheus) GetAlert(ctx context.Context, name string) (*AlertRule, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	rules, err := p.loadAlertRules(ctx)
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
func (p *Prometheus) PutAlert(ctx context.Context, rule *AlertRule) error {
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
	b, err := exec.Command(p.promtoolPath, "check-rules", f.Name()).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(b))
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// write to permanent location, reload configuration
	path := filepath.Join(p.alertRulesPath, rule.Name)
	path += ".rule"
	if rule.Disabled {
		path += ".disabled"
	}
	if err := ioutil.WriteFile(path, []byte(rule.Text), 0666); err != nil {
		return errors.WithStack(err)
	}
	return p.reload()
}

// DeleteAlert removes existing alert rule by name, or error if no such rule is present.
func (p *Prometheus) DeleteAlert(ctx context.Context, name string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	rules, err := p.loadAlertRules(ctx)
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
