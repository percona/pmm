// Copyright (C) 2024 Percona LLC
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

import "github.com/percona/pmm/version"

type versionResult struct{}

func (v versionResult) Result() {}

func (v versionResult) String() string {
	return version.FullInfo()
}

func (v versionResult) MarshalJSON() ([]byte, error) { //nolint:unparam
	return []byte(version.FullInfoJSON()), nil
}

// VersionCommand is used for CLI flags and commands.
type VersionCommand struct{}

// BeforeApply is run before the command is applied.
func (cmd *VersionCommand) BeforeApply() error {
	SetupClientsEnabled = false
	return nil
}

// RunCmd runs VersionCommand.
func (cmd *VersionCommand) RunCmd() (Result, error) {
	return versionResult{}, nil
}
