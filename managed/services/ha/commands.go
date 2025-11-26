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

package ha

import (
	"encoding/json"
	"fmt"
)

// CommandType represents the type of Raft command.
type CommandType string

const (
	// CommandTypeSetAgentConnection sets or updates an agent's connection status.
	CommandTypeSetAgentConnection CommandType = "set_agent_connection"
	// CommandTypeDeleteAgentConnection removes an agent's connection status.
	CommandTypeDeleteAgentConnection CommandType = "delete_agent_connection"
)

// Command represents a Raft log entry command.
type Command struct {
	Type    CommandType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// SetAgentConnectionPayload represents the payload for setting agent connection status.
type SetAgentConnectionPayload struct {
	AgentID   string `json:"agent_id"`
	Connected bool   `json:"connected"`
}

// DeleteAgentConnectionPayload represents the payload for deleting agent connection status.
type DeleteAgentConnectionPayload struct {
	AgentID string `json:"agent_id"`
}

// EncodeCommand encodes a command to JSON bytes.
func EncodeCommand(cmdType CommandType, payload any) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := Command{
		Type:    cmdType,
		Payload: payloadBytes,
	}

	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	return cmdBytes, nil
}

// DecodeCommand decodes a command from JSON bytes.
func DecodeCommand(data []byte) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %w", err)
	}
	return &cmd, nil
}
