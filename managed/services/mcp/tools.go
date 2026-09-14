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

package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// readOnly returns the annotations every PMM tool carries: read-only,
// non-destructive, idempotent, and closed-world (PMM only).
func readOnly(title string) *mcp.ToolAnnotations {
	f := false
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    true,
		DestructiveHint: &f,
		IdempotentHint:  true,
		OpenWorldHint:   &f,
	}
}

// textResult wraps text in a successful tool result.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// fail maps an error to the contract's typed error and logs it. Messages may
// carry PMM's own error text but never tokens or SQL.
func (s *Service) fail(tool string, err error) *toolError {
	te := mapError(err)
	s.l.WithField("tool", tool).WithField("code", string(te.code)).Warnf("Tool call failed: %s.", te.message)
	return te
}
