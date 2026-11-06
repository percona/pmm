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
	"context"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolTimeoutSlack is added to the action timeout to bound a whole tool call:
// the backing API calls have their own per-request deadline as well.
const toolTimeoutSlack = 45 * time.Second

// renderer is a tool result that renders itself as the text half of the
// result; the structured half is the value itself.
type renderer interface {
	render() string
}

// toolFunc is a tool implementation that returns a typed result.
type toolFunc[In any, Out renderer] func(ctx context.Context, req *mcp.CallToolRequest, in In) (Out, error)

// addTool registers a tool that returns its typed result as text and structuredContent.
//
// The SDK sends a typed Out's zero value next to isError, so tools register with Out = any and an explicit OutputSchema.
func addTool[In any, Out renderer](s *Service, server *mcp.Server, tool *mcp.Tool, fn toolFunc[In, Out]) {
	schema, err := jsonschema.For[Out](nil)
	if err != nil {
		panic("mcp: output schema of tool " + tool.Name + ": " + err.Error())
	}
	t := *tool
	t.OutputSchema = schema

	mcp.AddTool(server, &t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		l := s.l.WithField("tool", t.Name)
		start := time.Now()

		ctx, cancel := context.WithTimeout(ctx, s.actionTimeout()+toolTimeoutSlack)
		defer cancel()

		out, err := fn(ctx, req, in)
		if err != nil {
			te := mapError(err)
			l.WithField("code", string(te.code)).WithField("duration", time.Since(start)).Warn("Tool call failed.")
			// The message can quote SQL, so it is logged at Debug only.
			l.Debugf("Tool call failed: %s.", te.message)
			return errorResult(te), nil, nil
		}
		l.WithField("duration", time.Since(start)).Debug("Tool call succeeded.")
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: out.render()}}}, out, nil
	})
}

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
