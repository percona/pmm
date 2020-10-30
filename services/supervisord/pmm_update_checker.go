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

package supervisord

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	updateCheckInterval    = 24 * time.Hour
	updateCheckResultFresh = updateCheckInterval + 10*time.Minute
	updateDefaultTimeout   = 30 * time.Second
)

// PMMUpdateChecker wraps `pmm-update -installed` and `pmm-update -check` with caching.
//
// We almost could use `supervisorctl start pmm-update-check` and then get output from stdout log file,
// but that is too painful, and, unlike with `pmm-update -perform`, we don't have to do it.
type PMMUpdateChecker struct {
	l *logrus.Entry

	checkRW                  sync.RWMutex
	installedRW              sync.RWMutex
	cmdMutex                 sync.Mutex
	lastInstalledPackageInfo *version.PackageInfo
	lastCheckResult          *version.UpdateCheckResult
	lastCheckTime            time.Time
}

// NewPMMUpdateChecker returns a PMMUpdateChecker instance that can be shared across different parts of the code.
// Since this is used inside this package, it could be a singleton, but it would make things mode difficult to test.
func NewPMMUpdateChecker(l *logrus.Entry) *PMMUpdateChecker {
	return &PMMUpdateChecker{
		l: l,
	}
}

// run runs check for updates loop until ctx is canceled.
func (p *PMMUpdateChecker) run(ctx context.Context) {
	p.l.Info("Starting...")
	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		_ = p.check(ctx)

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			p.l.Info("Done.")
			return
		}
	}
}

// Installed returns currently installed version information.
// It is always cached since pmm-update RPM package is always updated before pmm-managed update/restart.
func (p *PMMUpdateChecker) Installed(ctx context.Context) *version.PackageInfo {
	p.installedRW.RLock()
	if p.lastInstalledPackageInfo != nil {
		res := p.lastInstalledPackageInfo
		p.installedRW.RUnlock()
		return res
	}
	p.installedRW.RUnlock()

	// use -installed since it is much faster
	cmdLine := "pmm-update -installed"
	b, stderr, err := p.cmdRun(ctx, cmdLine)
	if err != nil {
		p.l.Errorf("%s output: %s. Error: %s", cmdLine, stderr.Bytes(), err)
		return nil
	}

	var res version.UpdateInstalledResult
	if err = json.Unmarshal(b, &res); err != nil {
		p.l.Errorf("%s output: %s", cmdLine, stderr.Bytes())
		return nil
	}

	p.installedRW.Lock()
	p.lastInstalledPackageInfo = &res.Installed
	p.installedRW.Unlock()

	return &res.Installed
}

func (p *PMMUpdateChecker) cmdRun(ctx context.Context, cmdLine string) ([]byte, bytes.Buffer, error) {
	args := strings.Split(cmdLine, " ")
	p.cmdMutex.Lock()
	timeoutCtx, cancel := context.WithTimeout(ctx, updateDefaultTimeout)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, args[0], args[1:]...) //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err := cmd.Output()
	p.cmdMutex.Unlock()
	return b, stderr, err
}

// checkResult returns last `pmm-update -check` result and check time.
// It may force re-check if last result is empty or too old.
func (p *PMMUpdateChecker) checkResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	p.checkRW.RLock()
	defer p.checkRW.RUnlock()

	if time.Since(p.lastCheckTime) > updateCheckResultFresh {
		p.checkRW.RUnlock()
		_ = p.check(ctx)
		p.checkRW.RLock()
	}

	return p.lastCheckResult, p.lastCheckTime
}

// check calls `pmm-update -check` and fills lastInstalledPackageInfo/lastCheckResult/lastCheckTime on success.
func (p *PMMUpdateChecker) check(ctx context.Context) error {
	p.checkRW.Lock()
	defer p.checkRW.Unlock()

	cmdLine := "pmm-update -check"
	b, stderr, err := p.cmdRun(ctx, cmdLine)
	if err != nil {
		p.l.Errorf("%s output: %s. Error: %s", cmdLine, stderr.Bytes(), err)
		return errors.WithStack(err)
	}

	var res version.UpdateCheckResult
	if err = json.Unmarshal(b, &res); err != nil {
		p.l.Errorf("%s output: %s", cmdLine, stderr.Bytes())
		return errors.WithStack(err)
	}

	p.l.Debugf("%s output: %s", cmdLine, stderr.Bytes())
	p.installedRW.Lock()
	p.lastInstalledPackageInfo = &res.Installed
	p.installedRW.Unlock()
	p.lastCheckResult = &res
	p.lastCheckTime = time.Now()
	return nil
}
