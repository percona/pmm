// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"path/filepath"
	"strings"
)

const (
	tmpDataPathFlag = "remoteWrite.tmpDataPath"

	// What the kernel appends to /proc/<pid>/exe once the executable has been unlinked, which is
	// what an upgrade does to a vmagent binary that is still running from before it.
	deletedExeSuffix = " (deleted)"
)

// isVMAgentExecutable reports whether exe, a process's /proc/<pid>/exe, is this pmm-agent's vmagent
// binary, including one an upgrade has replaced since the process started.
func isVMAgentExecutable(exe, vmagentPath string) bool {
	if exe == "" || vmagentPath == "" {
		return false
	}

	return filepath.Clean(strings.TrimSuffix(exe, deletedExeSuffix)) == filepath.Clean(vmagentPath)
}

// tmpDataPath returns the -remoteWrite.tmpDataPath of a vmagent command line: the directory of the
// persistent queue it holds. The lock vmagent takes on that directory is exclusive, so while one
// process holds it, a vmagent started on the same queue exits on the lock within a second, every time.
func tmpDataPath(cmdline []string) (string, bool) {
	if len(cmdline) == 0 {
		return "", false
	}

	for _, arg := range cmdline[1:] { // not argv[0]
		flag := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-") // -flag or --flag
		if flag == arg {
			continue
		}
		if value, ok := strings.CutPrefix(flag, tmpDataPathFlag+"="); ok && value != "" {
			return value, true
		}
	}

	return "", false
}
