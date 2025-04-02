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

package vmalert

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/utils/validators"
)

const externalRulesFile = "/srv/prometheus/rules/pmm.rules.yml"

// ExternalRules contains all logic related to alerting rules files.
type ExternalRules struct {
	l *logrus.Entry
}

// NewExternalRules creates new ExternalRules instance.
func NewExternalRules() *ExternalRules {
	return &ExternalRules{
		l: logrus.WithField("component", "external_rules"),
	}
}

// ValidateRules validates alerting rules.
func (s *ExternalRules) ValidateRules(ctx context.Context, rules string) error {
	err := validators.ValidateAlertingRules(ctx, rules)
	if e, ok := err.(*validators.InvalidAlertingRuleError); ok { //nolint:errorlint
		return status.Error(codes.InvalidArgument, e.Msg)
	}
	return err
}

// ReadRules reads current rules from FS.
func (s *ExternalRules) ReadRules() (string, error) {
	b, err := os.ReadFile(externalRulesFile)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return string(b), nil
}

// RemoveRulesFile removes rules file from FS.
func (s *ExternalRules) RemoveRulesFile() error {
	return os.Remove(externalRulesFile)
}

// WriteRules writes rules to file.
func (s *ExternalRules) WriteRules(rules string) error {
	return os.WriteFile(externalRulesFile, []byte(rules), 0o644) //nolint:gosec
}
