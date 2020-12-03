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

package vmalert

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/validators"
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
	err := validators.ValidateAlertingRules(ctx, rules)
	if e, ok := err.(*validators.InvalidAlertingRuleError); ok {
		return status.Errorf(codes.InvalidArgument, e.Msg)
	}
	return err
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
	return ioutil.WriteFile(alertingRulesFile, []byte(rules), 0o644) //nolint:gosec
}
