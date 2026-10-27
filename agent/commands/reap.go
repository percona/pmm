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

// isLeftoverVMAgent reports whether a process is a vmagent holding this pmm-agent's persistent
// queue: it runs this pmm-agent's vmagent binary, and its -remoteWrite.tmpDataPath is this
// pmm-agent's queue directory. The lock vmagent takes on that directory is exclusive, so while such
// a process runs, the vmagent this pmm-agent starts exits on the lock within a second, every time.
//
// The parent is deliberately not consulted. Every exporter gets SIGKILL from pmm-agent as its
// parent-death signal, so a pmm-agent that is killed outright takes them with it; a vmagent that is
// still holding the queue when pmm-agent starts was left by something else, and nothing about its
// parent tells whether it is safe to keep. What it holds does: only one vmagent can use the queue.
func isLeftoverVMAgent(exe string, cmdline []string, vmagentPath, queueDir string) bool {
	if exe == "" || vmagentPath == "" || queueDir == "" || len(cmdline) < 2 {
		return false
	}

	if filepath.Clean(strings.TrimSuffix(exe, deletedExeSuffix)) != filepath.Clean(vmagentPath) {
		return false
	}

	for _, arg := range cmdline[1:] {
		flag := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-") // -flag or --flag
		if flag == arg {
			continue
		}
		value, ok := strings.CutPrefix(flag, tmpDataPathFlag+"=")
		if ok && filepath.Clean(value) == filepath.Clean(queueDir) {
			return true
		}
	}

	return false
}
