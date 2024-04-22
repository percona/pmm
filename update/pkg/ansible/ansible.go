// Copyright (C) 2024 Percona LLC
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

// Package ansible contains function for running playbooks.
package ansible

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/percona/pmm/update/pkg/run"
)

const ansibleCancelTimeout = 180 * time.Second // must be less than stopwaitsecs in supervisord config

// RunPlaybookOpts contains ansible-playbook options.
type RunPlaybookOpts struct {
	Debug      bool
	Trace      bool
	ExtraFlags []string
}

// RunPlaybook runs ansible-playbook.
func RunPlaybook(ctx context.Context, playbook string, opts *RunPlaybookOpts) error {
	if opts == nil {
		opts = &RunPlaybookOpts{}
	}

	var verbose string
	runOpts := &run.Opts{}
	if opts.Debug {
		verbose = "-vvv"
	}
	if opts.Trace {
		verbose = "-vvvv"
		runOpts.Env = []string{"ANSIBLE_DEBUG=1"}
	}

	cmdLine := fmt.Sprintf(
		`ansible-playbook --flush-cache %s %s %s`,
		verbose, strings.Join(opts.ExtraFlags, ""), playbook)

	_, _, err := run.Run(ctx, ansibleCancelTimeout, cmdLine, runOpts)
	return err
}
