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

package validators

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/utils/pdeathsig"
)

// InvalidAlertingRuleError represents "normal" alerting rule validation error.
type InvalidAlertingRuleError struct {
	Msg string
}

// Error implements error interface.
func (e *InvalidAlertingRuleError) Error() string {
	return e.Msg
}

// ValidateAlertingRules validates alerting rules (https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
// by storing them in temporary file and calling `vmalert -dryRun -rule`.
// Returned error is nil, *InvalidAlertingRuleError for "normal" validation errors,
// or some other fatal error.
func ValidateAlertingRules(ctx context.Context, rules string) error {
	tempFile, err := os.CreateTemp("", "temp_rules_*.yml")
	if err != nil {
		return fmt.Errorf("alerting rule validation failed: %w", err)
	}
	tempFile.Close()                 //nolint:errcheck
	defer os.Remove(tempFile.Name()) //nolint:errcheck

	err = os.WriteFile(tempFile.Name(), []byte(rules), 0o644) //nolint:gosec,mnd
	if err != nil {
		return fmt.Errorf("alerting rule validation failed: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second) //nolint:mnd
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "vmalert", "-loggerLevel", "WARN", "-dryRun", "-rule", tempFile.Name()) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err := cmd.CombinedOutput()
	logrus.Debugf("ValidateAlertingRules: %v\n%s", err, b)
	if err != nil {
		e, ok := errors.AsType[*exec.ExitError](err)
		if ok && e.ExitCode() != 0 {
			return &InvalidAlertingRuleError{
				Msg: "Invalid alerting rules.",
			}
		}
		return fmt.Errorf("alerting rule validation failed: %w", err)
	}

	return nil
}
