// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package prometheus

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const alertingRulesFile = "/srv/prometheus/rules/pmm.rules.yml"

// AlertingRules contains all logic related to alerting rules files.
type AlertingRules struct {
	l *logrus.Entry
}

// NewAlertingRules creates new AlertingRules instance.
func NewAlertingRules() *AlertingRules {
	return &AlertingRules{
		l: logrus.WithField("component", "alerting_rules"),
	}
}

// ValidateRules validates alerting rules.
func (s *AlertingRules) ValidateRules(ctx context.Context, rules string) error {
	tempFile, err := ioutil.TempFile("", "temp_rules_*.yml")
	if err != nil {
		return errors.WithStack(err)
	}
	tempFile.Close()                 //nolint:errcheck
	defer os.Remove(tempFile.Name()) //nolint:errcheck

	if err = ioutil.WriteFile(tempFile.Name(), []byte(rules), 0644); err != nil {
		return errors.WithStack(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// TODO use `vmalert -dryRun` https://jira.percona.com/browse/PMM-7011
	cmd := exec.CommandContext(timeoutCtx, "promtool", "check", "rules", tempFile.Name()) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err := cmd.CombinedOutput()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() != 0 {
			s.l.Infof("%s: %s\n%s", strings.Join(cmd.Args, " "), e, b)
			return status.Errorf(codes.InvalidArgument, "Invalid alerting rules.")
		}
		return errors.WithStack(err)
	}

	if bytes.Contains(b, []byte("SUCCESS: 0 rules found")) {
		return status.Errorf(codes.InvalidArgument, "Zero alerting rules found.")
	}

	s.l.Debugf("%q check passed.", strings.Join(cmd.Args, " "))
	return nil
}

// ReadRules reads current rules from FS.
func (s *AlertingRules) ReadRules() (string, error) {
	b, err := ioutil.ReadFile(alertingRulesFile)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return string(b), nil
}

// RemoveRulesFile removes rules file from FS.
func (s *AlertingRules) RemoveRulesFile() error {
	return os.Remove(alertingRulesFile)
}

// WriteRules writes rules to file.
func (s *AlertingRules) WriteRules(rules string) error {
	return ioutil.WriteFile(alertingRulesFile, []byte(rules), 0644)
}
