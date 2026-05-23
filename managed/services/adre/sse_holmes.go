// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Holmes SSE event names (holmes/utils/stream.py).
const (
	holmesSSEEventAnswerEnd        = "ai_answer_end"
	holmesSSEEventError            = "error"
	holmesSSEEventToolResult       = "tool_calling_result"
	holmesSSEEventApprovalRequired = "approval_required"
)

// holmesStreamOutcome is the parsed result of a Holmes streaming response.
type holmesStreamOutcome struct {
	Analysis           string
	ToolResultJSONRows [][]byte
	Metadata           json.RawMessage
	PromptTokens       *int32
	CompletionTokens   *int32
	TotalTokens        *int32
}

// parseHolmesSSEStream tees every line to forward (if non-nil), then parses events for persistence.
func parseHolmesSSEStream(src io.Reader, forward func([]byte) error) (out holmesStreamOutcome, sawErrorEvent bool, err error) { //nolint:gocognit,nonamedreturns
	sc := bufio.NewScanner(src)
	// Large Holmes payloads in a single data: line.
	sc.Buffer(make([]byte, 64*1024), 1024*1024) //nolint:mnd

	var eventName string
	var dataLines []string

	dispatch := func() {
		if eventName == "" && len(dataLines) == 0 {
			return
		}
		ev := eventName
		payload := strings.Join(dataLines, "\n")
		eventName = ""
		dataLines = dataLines[:0]
		if strings.TrimSpace(payload) == "" {
			return
		}
		raw := []byte(payload)
		switch ev {
		case holmesSSEEventError:
			sawErrorEvent = true
		case holmesSSEEventToolResult:
			out.ToolResultJSONRows = append(out.ToolResultJSONRows, append([]byte(nil), raw...))
		case holmesSSEEventAnswerEnd, holmesSSEEventApprovalRequired:
			var d struct {
				Analysis string          `json:"analysis"`
				Metadata json.RawMessage `json:"metadata"`
			}
			err := json.Unmarshal(raw, &d)
			if err != nil {
				return
			}
			out.Analysis = d.Analysis
			if len(d.Metadata) > 0 {
				out.Metadata = append(json.RawMessage(nil), d.Metadata...)
				usage := ParseHolmesMetadata(d.Metadata)
				if usage != nil {
					out.PromptTokens = usage.PromptTokens
					out.CompletionTokens = usage.CompletionTokens
					out.TotalTokens = usage.TotalTokens
				}
			}
		}
	}

	for sc.Scan() {
		line := sc.Bytes()
		if forward != nil {
			err := forward(append(append([]byte(nil), line...), '\n'))
			if err != nil {
				return out, sawErrorEvent, err
			}
		}
		if bytes.HasPrefix(line, []byte("event:")) {
			dispatch()
			eventName = strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("event:"))))
			continue
		}
		if after, ok := bytes.CutPrefix(line, []byte("data:")); ok {
			dataLines = append(dataLines, strings.TrimSpace(string(after)))
			continue
		}
		if len(bytes.TrimSpace(line)) == 0 {
			dispatch()
		}
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) { //nolint:noinlineerr
		return out, sawErrorEvent, err
	}
	dispatch()
	return out, sawErrorEvent, nil
}
