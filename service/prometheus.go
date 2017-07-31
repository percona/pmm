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

package service

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/config"

	"github.com/Percona-Lab/pmm-managed/utils/logger"
)

type AlertRule struct {
	Name     string
	FilePath string
	Text     string
	Disabled bool
}

// Prometheus service is responsible for interaction with Prometheus process on the same host.
// It assumes the following about its configuration:
//   * TODO
type Prometheus struct {
	ConfigPath     string
	URL            *url.URL
	AlertRulesPath string
	PromtoolPath   string
	lock           sync.RWMutex
}

// loadConfig returns current Prometheus configuration.
func (p *Prometheus) loadConfig() (*config.Config, error) {
	c, err := config.LoadFile(p.ConfigPath)
	err = errors.Wrap(err, "can't load Prometheus configuration file")

	if len(c.RuleFiles) == 0 {
		return nil, errors.New("no RuleFiles patterns")
	}
	// TODO check p.AlertRulesPath is in c.RuleFiles patterns

	return c, err
}

// reload causes Prometheus to reload configuration, including alert rules files.
// Does nothing if Prometheus URL is not set.
func (p *Prometheus) reload() error {
	u := *p.URL
	u.Path = filepath.Join(u.Path, "-", "reload")
	resp, err := http.Post(u.String(), "", nil)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.Errorf("%d: %s", resp.StatusCode, b)
}

// loadAlertRules returns all Prometheus alert rules.
func (p *Prometheus) loadAlertRules(ctx context.Context) ([]AlertRule, error) {
	files, err := filepath.Glob(filepath.Join(p.AlertRulesPath, "*"))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	names := make(map[string]struct{})
	rules := make([]AlertRule, 0, len(files))
	for _, f := range files {
		// extract rule name and disabled status from filename
		var disabled bool
		base := filepath.Base(f)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		if ext == ".disabled" {
			disabled = true
			ext = filepath.Ext(name)
			name = strings.TrimSuffix(name, ext)
		}
		if ext != ".rule" {
			logger.Get(ctx).Warnf("unexpected file %q, skipped", f)
			continue
		}
		if _, ok := names[name]; ok {
			return nil, errors.Errorf("duplicate alert rule name %q", name)
		}
		names[name] = struct{}{}

		// load file and make rule
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		rule := AlertRule{
			Name:     name,
			FilePath: f,
			Text:     string(b),
			Disabled: disabled,
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (p *Prometheus) Check(ctx context.Context) error {
	if _, err := p.loadConfig(); err != nil {
		return err
	}

	if p.URL == nil {
		return errors.New("URL is not set")
	}
	u := *p.URL
	u.Path = filepath.Join(u.Path, "version")
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	logger.Get(ctx).Infof("Prometheus version: %s", b)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}
	return nil
}

// ListAlertRules returns all alert rules.
func (p *Prometheus) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.loadAlertRules(ctx)
}

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
	b, err := exec.Command(p.PromtoolPath, "check-rules", f.Name()).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(b))
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// write to permanent location, reload configuration
	path := filepath.Join(p.AlertRulesPath, rule.Name)
	path += ".rule"
	if rule.Disabled {
		path += ".disabled"
	}
	if err := ioutil.WriteFile(path, []byte(rule.Text), 0666); err != nil {
		return errors.WithStack(err)
	}
	return p.reload()
}

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
