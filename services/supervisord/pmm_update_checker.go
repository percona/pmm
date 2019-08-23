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
)

// pmmUpdateChecker wraps `pmm2-update -installed` and `pmm2-update -check` with caching.
//
// We almost could use `supervisorctl start pmm-update-check` and then get output from stdout log file,
// but that is too painful, and, unlike with `pmm2-update -perform`, we don't have to do it.
type pmmUpdateChecker struct {
	l *logrus.Entry

	rw                       sync.RWMutex
	lastInstalledPackageInfo *version.PackageInfo
	lastCheckResult          *version.UpdateCheckResult
	lastCheckTime            time.Time
}

func newPMMUpdateChecker(l *logrus.Entry) *pmmUpdateChecker {
	return &pmmUpdateChecker{
		l: l,
	}
}

// run runs check for updates loop until ctx is canceled.
func (p *pmmUpdateChecker) run(ctx context.Context) {
	p.l.Info("Starting...")
	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		_ = p.check()

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			p.l.Info("Done.")
			return
		}
	}
}

// installed returns currently installed version information.
// It is always cached since pmm-update RPM package is always updated before pmm-managed update/restart.
func (p *pmmUpdateChecker) installed() *version.PackageInfo {
	p.rw.RLock()
	if p.lastInstalledPackageInfo != nil {
		res := p.lastInstalledPackageInfo
		p.rw.RUnlock()
		return res
	}
	p.rw.RUnlock()

	// use -installed since it is much faster
	cmdLine := "pmm2-update -installed"
	args := strings.Split(cmdLine, " ")
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err := cmd.Output()
	if err != nil {
		p.l.Errorf("%s output: %s. Error: %s", cmdLine, stderr.Bytes(), err)
		return nil
	}

	var res version.UpdateInstalledResult
	if err = json.Unmarshal(b, &res); err != nil {
		p.l.Errorf("%s output: %s", cmdLine, stderr.Bytes())
		return nil
	}

	p.rw.Lock()
	p.lastInstalledPackageInfo = &res.Installed
	p.rw.Unlock()

	return &res.Installed
}

// checkResult returns last `pmm-update -check` result and check time.
// It may force re-check if last result is empty or too old.
func (p *pmmUpdateChecker) checkResult() (*version.UpdateCheckResult, time.Time) {
	p.rw.RLock()
	defer p.rw.RUnlock()

	if time.Since(p.lastCheckTime) > updateCheckResultFresh {
		p.rw.RUnlock()
		_ = p.check()
		p.rw.RLock()
	}

	return p.lastCheckResult, p.lastCheckTime
}

// check calls `pmm2-update -check` and fills lastInstalledPackageInfo/lastCheckResult/lastCheckTime on success.
func (p *pmmUpdateChecker) check() error {
	p.rw.Lock()
	defer p.rw.Unlock()

	cmdLine := "pmm2-update -check"
	args := strings.Split(cmdLine, " ")
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err := cmd.Output()
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
	p.lastInstalledPackageInfo = &res.Installed
	p.lastCheckResult = &res
	p.lastCheckTime = time.Now()
	return nil
}
