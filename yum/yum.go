// pmm-update
// Copyright (C) 2019 Percona LLC
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

package yum

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// run runs command and returns stdout and stderr lines.
// Both are also tee'd to os.Stderr for a progress reporting.
func run(ctx context.Context, cmdLine string) ([]string, []string, error) {
	// TODO when ctx is canceled, send SIGTERM, wait X seconds, and _then_ send SIGKILL;
	// CommandContext sends SIGKILL as soon as ctx is canceled, do not use it

	args := strings.Fields(cmdLine)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	setSysProcAttr(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stderr, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	err := cmd.Run()
	return strings.Split(stdout.String(), "\n"), strings.Split(stderr.String(), "\n"), err
}

// Versions contains information about RPM package versions.
type Versions struct {
	Installed  string `json:"installed"`
	Remote     string `json:"remote"`
	RemoteRepo string `json:"remote_repo"`
}

// CheckVersions returns up-to-date versions information for a package with given name.
func CheckVersions(ctx context.Context, name string) (*Versions, error) {
	// http://man7.org/linux/man-pages/man8/yum.8.html#LIST_OPTIONS

	stdout, _, err := run(ctx, "yum --showduplicates list all "+name)
	if err != nil {
		return nil, errors.Wrap(err, "`yum list` failed")
	}

	var res Versions
	for _, line := range stdout {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) != 3 {
			continue
		}
		pack, ver, repo := parts[0], parts[1], parts[2]

		if !strings.HasPrefix(pack, name+".") {
			continue
		}
		if strings.HasPrefix(repo, "@") {
			if res.Installed != "" {
				return nil, errors.New("failed to parse `yum list` output")
			}
			res.Installed = ver
		} else {
			// always overwrite - the last item is the one we need
			res.Remote = ver
			res.RemoteRepo = repo
		}
	}

	return &res, nil
}
