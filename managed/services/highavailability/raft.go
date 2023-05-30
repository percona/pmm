package highavailability

import (
	"context"
	"io"
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/sirupsen/logrus"
)

// raftFSM is a simple example implementation of a Raft FSM.
type raftFSM struct {
	m        *memberlist.Memberlist
	mu       sync.Mutex
	nodeID   raft.ServerID
	leaderID raft.ServerID

	l *logrus.Entry
}

func newFSM(m *memberlist.Memberlist, nodeID raft.ServerID) *raftFSM {
	return &raftFSM{
		m:      m,
		nodeID: nodeID,
		l:      logrus.WithField("component", "raftFSM"),
	}
}

func (f *raftFSM) Run(ctx context.Context, leaderCh chan raft.Observation) {
	for {
		select {
		case o := <-leaderCh:
			l := o.Data.(raft.LeaderObservation)
			f.LeaderChange(l.LeaderID)
		case <-ctx.Done():
			return
		}
	}
}

func (f *raftFSM) Apply(logEntry *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.l.Printf("raft: got a message: %s", string(logEntry.Data))
	// If this node is the leader, broadcast the message to all the other nodes
	if f.nodeID == f.leaderID {
		f.l.Printf("Broadcasting message: %s", string(logEntry.Data))
		for _, peer := range f.m.Members() {
			if peer.Name == string(f.nodeID) {
				continue
			}
			f.l.Printf("Sending message to: %s", peer.Name)
			err := f.m.SendReliable(peer, logEntry.Data)
			if err != nil {
				f.l.Printf("Failed to send message to node %s: %v", peer.Name, err)
			}
		}
	}

	return nil
}

func (f *raftFSM) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

func (f *raftFSM) Restore(rc io.ReadCloser) error {
	return nil
}

func (f *raftFSM) LeaderChange(leader raft.ServerID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.l.Printf("Leader changed from %s to %s", f.leaderID, leader)
	f.leaderID = leader
}
