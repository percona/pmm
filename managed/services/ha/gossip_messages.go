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
	"time"
)

// GossipMessageType represents the type of gossip message.
type GossipMessageType string

const (
	// GossipAgentConnect indicates an agent connection event.
	GossipAgentConnect GossipMessageType = "agent_connect"
	// GossipAgentDisconnect indicates an agent disconnection event.
	GossipAgentDisconnect GossipMessageType = "agent_disconnect"
	// GossipFullStateSync indicates a full state synchronization message.
	GossipFullStateSync GossipMessageType = "full_state_sync"
)

// GossipMessage represents a message broadcasted via gossip protocol.
type GossipMessage struct {
	Type      GossipMessageType `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      json.RawMessage   `json:"data"`
}

// AgentConnectionEvent represents an agent connection or disconnection event.
type AgentConnectionEvent struct {
	AgentID   string    `json:"agent_id"`
	ServerID  string    `json:"server_id"`
	EventType string    `json:"event_type"` // "connect" or "disconnect"
	Timestamp time.Time `json:"timestamp"`
}

// FullStateSyncRequest requests full agent location state from another server.
type FullStateSyncRequest struct {
	RequesterServerID string `json:"requester_server_id"`
}

// FullStateSyncResponse contains full agent location state.
type FullStateSyncResponse struct {
	AgentLocations map[string]string `json:"agent_locations"` // agent_id -> server_id
	Timestamp      time.Time         `json:"timestamp"`
}

// SerializeGossipMessage serializes a gossip message to bytes for broadcasting.
func SerializeGossipMessage(msgType GossipMessageType, data interface{}) ([]byte, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	msg := GossipMessage{
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      dataBytes,
	}

	return json.Marshal(msg)
}

// DeserializeGossipMessage deserializes bytes into a gossip message.
func DeserializeGossipMessage(data []byte) (*GossipMessage, error) {
	var msg GossipMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseAgentConnectionEvent parses agent connection event data from gossip message.
func ParseAgentConnectionEvent(data json.RawMessage) (*AgentConnectionEvent, error) {
	var event AgentConnectionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseFullStateSyncResponse parses full state sync response from gossip message.
func ParseFullStateSyncResponse(data json.RawMessage) (*FullStateSyncResponse, error) {
	var response FullStateSyncResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

