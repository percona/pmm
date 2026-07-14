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

// Package run contains function for running commands.
package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/utils/pdeathsig"
)

// Opts contains run options.
type Opts struct {
	Env []string
}

// Run runs command and returns stdout and stderr lines. Both are also tee'd to os.Stderr for a progress reporting.
// When ctx is canceled, SIGTERM is sent, and then SIGKILL after cancelTimeout.
func Run(ctx context.Context, cancelTimeout time.Duration, cmdLine string, opts *Opts) ([]string, []string, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	if opts == nil {
		opts = &Opts{}
	}
	if opts.Env == nil {
		// inherit environment, not overwrite it
		opts.Env = append(os.Environ(), opts.Env...)
	}

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	args := strings.Fields(cmdLine)
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...) //nolint:gosec,contextcheck

	// restrict process
	cmd.Env = opts.Env
	cmd.Dir = "/"
	pdeathsig.Set(cmd, unix.SIGKILL)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stderr, &stdout) // stdout to stderr
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	fmt.Fprintf(os.Stderr, "Starting %q ...\n", strings.Join(cmd.Args, " "))
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	go func() {
		select {
		case <-cmdCtx.Done():
		case <-ctx.Done():
			if err := cmd.Process.Signal(unix.SIGTERM); err != nil {
				logrus.Warnf("Failed to send SIGTERM.")
			}
			t := time.AfterFunc(cancelTimeout, cancel)
			defer t.Stop()
		}
	}()

	err := cmd.Wait()
	stdoutS := strings.Split(stdout.String(), "\n")
	stderrS := strings.Split(stderr.String(), "\n")
	return stdoutS, stderrS, err
}
