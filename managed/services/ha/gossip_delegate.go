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
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/sirupsen/logrus"
)

// gossipDelegate implements memberlist.Delegate for custom gossip messages.
type gossipDelegate struct {
	service *Service
	l       *logrus.Entry

	broadcastQueue *memberlist.TransmitLimitedQueue
	mu             sync.RWMutex
}

// newGossipDelegate creates a new gossip delegate.
func newGossipDelegate(service *Service) *gossipDelegate {
	d := &gossipDelegate{
		service: service,
		l:       logrus.WithField("component", "ha-gossip-delegate"),
	}

	// Create broadcast queue with reasonable limits
	d.broadcastQueue = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			if service.memberlist == nil {
				return 1
			}
			return service.memberlist.NumMembers()
		},
		RetransmitMult: 3, // Retransmit 3 times for reliability
	}

	return d
}

// NodeMeta is used to retrieve meta-data about the current node
// when broadcasting an alive message. It's length is limited to
// the given byte size. This metadata is available in the Node structure.
func (d *gossipDelegate) NodeMeta(limit int) []byte {
	// Return empty metadata for now
	// Could include server version, agent count, etc. in future
	return []byte{}
}

// NotifyMsg is called when a user-data message is received.
// Care should be taken that this method does not block, since doing
// so would block the entire UDP packet receive loop. Additionally, the byte
// slice may be modified after the call returns, so it should be copied if needed.
func (d *gossipDelegate) NotifyMsg(data []byte) {
	// Make a copy since the slice may be reused
	msgData := make([]byte, len(data))
	copy(msgData, data)

	// Parse the gossip message
	msg, err := DeserializeGossipMessage(msgData)
	if err != nil {
		d.l.Warnf("Failed to deserialize gossip message: %v", err)
		return
	}

	d.l.Debugf("Received gossip message: type=%s timestamp=%s", msg.Type, msg.Timestamp)

	// Handle based on message type
	switch msg.Type {
	case GossipAgentConnect, GossipAgentDisconnect:
		d.handleAgentConnectionEvent(msg)
	case GossipFullStateSync:
		d.handleFullStateSync(msg)
	default:
		d.l.Warnf("Unknown gossip message type: %s", msg.Type)
	}
}

// handleAgentConnectionEvent processes agent connection/disconnection events.
func (d *gossipDelegate) handleAgentConnectionEvent(msg *GossipMessage) {
	event, err := ParseAgentConnectionEvent(msg.Data)
	if err != nil {
		d.l.Warnf("Failed to parse agent connection event: %v", err)
		return
	}

	d.l.Debugf("Agent connection event: agent=%s server=%s event=%s",
		event.AgentID, event.ServerID, event.EventType)

	// Update agent location map
	d.service.updateAgentLocation(event.AgentID, event.ServerID, event.EventType)
}

// handleFullStateSync processes full state synchronization messages.
func (d *gossipDelegate) handleFullStateSync(msg *GossipMessage) {
	response, err := ParseFullStateSyncResponse(msg.Data)
	if err != nil {
		d.l.Warnf("Failed to parse full state sync response: %v", err)
		return
	}

	d.l.Infof("Received full state sync: %d agents", len(response.AgentLocations))

	// Merge state into local agent locations
	d.service.mergeAgentLocations(response.AgentLocations)
}

// GetBroadcasts is called when user data messages can be broadcast.
// It can return a list of buffers to send. Each buffer should assume an
// overhead as provided with a limit on the total byte size allowed.
// The total byte size of the resulting data to send must not exceed
// the limit. Care should be taken that this method does not block,
// since doing so would block the entire UDP packet receive loop.
func (d *gossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.broadcastQueue == nil {
		return nil
	}

	return d.broadcastQueue.GetBroadcasts(overhead, limit)
}

// LocalState is used for a TCP Push/Pull. This is sent to
// the remote side in addition to the membership information. Any
// data can be sent here. See MergeRemoteState as well. The `join`
// boolean indicates this is for a join instead of a push/pull.
func (d *gossipDelegate) LocalState(join bool) []byte {
	// Return full agent location state for TCP push/pull
	d.l.Debugf("LocalState requested, join=%v", join)

	state := d.service.getFullAgentState()
	data, err := SerializeGossipMessage(GossipFullStateSync, state)
	if err != nil {
		d.l.Errorf("Failed to serialize local state: %v", err)
		return nil
	}

	return data
}

// MergeRemoteState is invoked after a TCP Push/Pull. This is the
// state received from the remote side and is the result of the
// remote side's LocalState call. The 'join'
// boolean indicates this is for a join instead of a push/pull.
func (d *gossipDelegate) MergeRemoteState(buf []byte, join bool) {
	d.l.Debugf("MergeRemoteState called, join=%v, size=%d", join, len(buf))

	msg, err := DeserializeGossipMessage(buf)
	if err != nil {
		d.l.Warnf("Failed to deserialize remote state: %v", err)
		return
	}

	if msg.Type != GossipFullStateSync {
		d.l.Warnf("Unexpected message type in MergeRemoteState: %s", msg.Type)
		return
	}

	response, err := ParseFullStateSyncResponse(msg.Data)
	if err != nil {
		d.l.Warnf("Failed to parse remote state: %v", err)
		return
	}

	d.l.Infof("Merging remote state: %d agents", len(response.AgentLocations))
	d.service.mergeAgentLocations(response.AgentLocations)
}

// queueBroadcast queues a message for broadcasting via gossip.
func (d *gossipDelegate) queueBroadcast(data []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.broadcastQueue == nil {
		d.l.Warn("Broadcast queue not initialized")
		return
	}

	broadcast := &gossipBroadcast{
		msg:    data,
		notify: nil,
	}

	d.broadcastQueue.QueueBroadcast(broadcast)
}

// gossipBroadcast implements memberlist.Broadcast for queuing messages.
type gossipBroadcast struct {
	msg    []byte
	notify chan<- struct{}
}

// Invalidates implements memberlist.Broadcast.
func (b *gossipBroadcast) Invalidates(other memberlist.Broadcast) bool {
	// Agent connection events don't invalidate each other
	return false
}

// Message implements memberlist.Broadcast.
func (b *gossipBroadcast) Message() []byte {
	return b.msg
}

// Finished implements memberlist.Broadcast.
func (b *gossipBroadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}

// check interface implementation.
var _ memberlist.Delegate = (*gossipDelegate)(nil)

