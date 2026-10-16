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

package otel

import (
	"regexp"
	"strings"
)

// IndentYAML prefixes each line of block with indent so it can be nested under a YAML key
// (e.g. filelog receiver operators under "operators:").
func IndentYAML(block, indent string) string {
	if block == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(block, "\n"), "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n") + "\n"
}

// sanitizePresetName returns a safe receiver id suffix (alphanumeric and underscore only).
func sanitizePresetName(name string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(name, "_")
}

// FilelogMultilineYAML returns a filelog receiver multiline block that starts a new entry at lines matching
// lineStartPattern (so continuation lines join the entry), or "" when the pattern is empty.
func FilelogMultilineYAML(lineStartPattern, indent string) string {
	if lineStartPattern == "" {
		return ""
	}
	return indent + "multiline:\n" + indent + "  line_start_pattern: " + quoteYAMLString(lineStartPattern) + "\n"
}

func quoteYAMLString(s string) string {
	if s == "" || (!strings.Contains(s, "'") && !strings.Contains(s, "\n")) {
		return "'" + s + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
