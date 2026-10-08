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

package inventory

import "github.com/percona/pmm/admin/pkg/flags"

// Helper function to convert log level pointer to API enum for change commands.
func convertLogLevelPtr(level *flags.LogLevel) *string {
	if level == nil {
		return nil
	}

	return level.EnumValue()
}

// appendToggleChange appends onMsg or offMsg to changes when the boolean flag was provided.
func appendToggleChange(changes []string, flag *bool, onMsg, offMsg string) []string {
	if flag == nil {
		return changes
	}
	if *flag {
		return append(changes, onMsg)
	}

	return append(changes, offMsg)
}
