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

// Package ha contains everything related to high availability.
package ha

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

const (
	defaultNodeEventChanSize     = 5
	defaultRaftRetries           = 3
	defaultTransportTimeout      = 10 * time.Second
	defaultLeaveTimeout          = 5 * time.Second
	defaultLeaveAnnounceTimeout  = 2 * time.Second
	defaultObservationChanSize   = 16
	defaultRaftProbeTimeout      = time.Second
	defaultRaftLogRepeatInterval = time.Minute
	defaultTickerInterval        = 5 * time.Second
	defaultApplyTimeout          = 3 * time.Second
	defaultRaftDataDir           = "/srv/ha"
	defaultRaftDataDirPerm       = 0o750
	defaultSnapshotRetention     = 3
	defaultSnapshotThreshold     = 8192
	defaultTrailingLogs          = 10240
	defaultHeartbeatTimeout      = 1000 * time.Millisecond
	defaultElectionTimeout       = 1000 * time.Millisecond
	defaultCommitTimeout         = 50 * time.Millisecond
	defaultLeaderLeaseTimeout    = 500 * time.Millisecond
	defaultSnapshotInterval      = 120 * time.Second
	defaultServerOpTimeout       = 10 * time.Second
	defaultDNSLookupTimeout      = 3 * time.Second
	defaultRejoinInterval        = 30 * time.Second
	defaultRejoinDialTimeout     = 2 * time.Second
)

// memberlistMetaLeaving is the memberlist metadata a node gossips while it shuts down gracefully.
const memberlistMetaLeaving = "leaving"

// Service represents the high-availability service.
type Service struct {
	params           *models.HAParams
	bootstrapCluster bool

	services *services

	nodeCh   chan memberlist.NodeEvent
	leaderCh chan raft.Observation
	delegate *memberlistDelegate
	// raftReachable checks a Raft address before a removal, see liveVoters.
	raftReachable func(ctx context.Context, address raft.ServerAddress) bool

	l  *logrus.Entry
	wg *sync.WaitGroup

	rw         sync.RWMutex
	raftNode   *raft.Raft
	memberlist *memberlist.Memberlist

	// peersMu guards left, the nodes that announced a graceful leave and haven't rejoined since,
	// and unreachable, the voters this node failed to heartbeat as the leader.
	peersMu     sync.Mutex
	left        map[string]struct{}
	unreachable map[raft.ServerID]struct{}
}

// memberlistDelegate lets other nodes tell a graceful leave from a failure: on shutdown, this node's metadata
// becomes memberlistMetaLeaving before it leaves, and memberlist passes that metadata with the leave event.
type memberlistDelegate struct {
	leaving atomic.Bool
}

// NodeMeta implements memberlist.Delegate.
func (d *memberlistDelegate) NodeMeta(int) []byte {
	if d.leaving.Load() {
		return []byte(memberlistMetaLeaving)
	}
	return nil
}

// NotifyMsg implements memberlist.Delegate.
func (d *memberlistDelegate) NotifyMsg([]byte) {}

// GetBroadcasts implements memberlist.Delegate.
func (d *memberlistDelegate) GetBroadcasts(int, int) [][]byte { return nil }

// LocalState implements memberlist.Delegate.
func (d *memberlistDelegate) LocalState(bool) []byte { return nil }

// MergeRemoteState implements memberlist.Delegate.
func (d *memberlistDelegate) MergeRemoteState([]byte, bool) {}

var _ memberlist.Delegate = (*memberlistDelegate)(nil)

// Apply applies a log entry to the high-availability service.
// Currently only used for Raft consensus, not for state replication.
func (s *Service) Apply(logEntry *raft.Log) any {
	s.l.Debugf("raft: applied log entry: index=%d, data=%s", logEntry.Index, string(logEntry.Data))
	return nil
}

// Snapshot returns a snapshot of the high-availability service.
// Since PMM HA uses Raft for leader election only (not state replication),
// the FSM has no state to snapshot. Cluster configuration (voters) is
// automatically stored by Raft in the snapshot metadata.
func (s *Service) Snapshot() (raft.FSMSnapshot, error) { //nolint:ireturn
	return &fsmSnapshot{}, nil
}

// Restore restores the high availability service to a previous state.
// Since PMM HA is stateless (leader election only), there's nothing to restore.
// Cluster configuration (voters) is automatically restored by Raft from snapshot metadata.
func (s *Service) Restore(rc io.ReadCloser) error {
	// FSM has no state, but we need to consume the reader
	// Raft automatically restores cluster configuration from metadata
	s.l.Debug("Restore called - FSM is stateless, cluster config restored by Raft")
	return rc.Close()
}

// fsmSnapshot implements raft.FSMSnapshot for stateless PMM HA.
type fsmSnapshot struct{}

// Persist writes an empty snapshot since PMM HA FSM is stateless.
// Cluster configuration (voters, etc.) is automatically persisted by Raft in metadata.
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	return sink.Close()
}

// Release is called when we are finished with the snapshot.
func (f *fsmSnapshot) Release() {
	// Nothing to release for stateless FSM
}

// memberlistLogWriter is an io.Writer that converts memberlist's standard log format to structured output.
type memberlistLogWriter struct {
	logger   *logrus.Entry
	logRegex *regexp.Regexp
}

// newMemberlistLogWriter creates a new log writer for memberlist.
func newMemberlistLogWriter(logger *logrus.Entry) *memberlistLogWriter {
	return &memberlistLogWriter{
		logger:   logger,
		logRegex: regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[(\w+)\] (?:memberlist: )?(.+)$`),
	}
}

// Write implements io.Writer interface and converts memberlist logs to logrus format.
func (w *memberlistLogWriter) Write(p []byte) (int, error) {
	// Remove trailing newline for parsing
	msg := string(bytes.TrimRight(p, "\n"))

	// Parse memberlist log format: "2025/12/22 21:43:27 [DEBUG|INFO|WARN|ERR] message"
	matches := w.logRegex.FindStringSubmatch(msg)
	if len(matches) == 3 { //nolint:mnd
		level := strings.ToLower(matches[1])
		message := matches[2]

		// Log with appropriate level
		switch level {
		case "debug":
			w.logger.Debug(message)
		case "info":
			w.logger.Info(message)
		case "warn":
			w.logger.Warn(message)
		case "err":
			w.logger.Error(message)
		default:
			w.logger.Info(message)
		}
	} else {
		// Fallback for unparseable logs
		w.logger.Info(msg)
	}

	return len(p), nil
}

var _ io.Writer = (*memberlistLogWriter)(nil)

// raftPeerFailure matches Raft's per-peer RPC failures, which repeat for as long as a voter is down.
var raftPeerFailure = regexp.MustCompile(`raft: failed to (?:heartbeat|appendEntries) to: peer=("[^"]*"|\S+)`)

// raftLogLimiter writes Raft's log lines through, but each per-peer RPC failure at most once per interval:
// a voter that refuses connections otherwise logs about two errors a second while it is down.
type raftLogLimiter struct {
	out      io.Writer
	interval time.Duration

	mu     sync.Mutex
	logged map[string]time.Time
}

func newRaftLogLimiter(out io.Writer, interval time.Duration) *raftLogLimiter {
	return &raftLogLimiter{out: out, interval: interval, logged: make(map[string]time.Time)}
}

// Write implements io.Writer; hclog writes each log line with a single call.
func (w *raftLogLimiter) Write(p []byte) (int, error) {
	match := raftPeerFailure.Find(p)
	if match != nil {
		w.mu.Lock()
		last, ok := w.logged[string(match)]
		if ok && time.Since(last) < w.interval {
			w.mu.Unlock()
			return len(p), nil
		}
		w.logged[string(match)] = time.Now()
		w.mu.Unlock()
	}
	return w.out.Write(p)
}

var _ io.Writer = (*raftLogLimiter)(nil)

// setupRaftStorage sets up persistent storage for Raft.
func setupRaftStorage(nodeID string, l *logrus.Entry) (*raftboltdb.BoltStore, *raftboltdb.BoltStore, *raft.FileSnapshotStore, error) {
	// Create the Raft data directory for this node
	raftDir := filepath.Join(defaultRaftDataDir, nodeID)
	err := os.MkdirAll(raftDir, defaultRaftDataDirPerm)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Raft data directory: %w", err)
	}
	l.Infof("Using Raft data directory: %s", raftDir)

	// Create BoltDB-based log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BoltDB log store: %w", err)
	}

	// Create BoltDB-based stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
	if err != nil {
		cerr := logStore.Close()
		if cerr != nil {
			l.Errorf("failed to close logStore after stableStore error: %v", cerr)
		}
		return nil, nil, nil, fmt.Errorf("failed to create BoltDB stable store: %w", err)
	}

	// Create file-based snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, defaultSnapshotRetention, os.Stderr)
	if err != nil {
		cerr := logStore.Close()
		if cerr != nil {
			l.Errorf("failed to close logStore after snapshotStore error: %v", cerr)
		}
		cerr = stableStore.Close()
		if cerr != nil {
			l.Errorf("failed to close stableStore after snapshotStore error: %v", cerr)
		}
		return nil, nil, nil, fmt.Errorf("failed to create file snapshot store: %w", err)
	}

	return logStore, stableStore, snapshotStore, nil
}

// bootstrapsCluster reports whether this node bootstraps the Raft cluster: only the first of PMM_HA_PEERS, which then adds
// the others, so that no node starts a cluster of its own, unless the peers aren't named <node ID>.<domain> as the chart does.
func bootstrapsCluster(params *models.HAParams) bool {
	ids := make([]string, 0, len(params.Nodes))
	for _, peer := range params.Nodes {
		id, ok := models.HAPeerNodeName(peer)
		if !ok {
			return true
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return true
	}
	if slices.Contains(ids, params.NodeID) {
		return ids[0] == params.NodeID
	}
	// kubectl scale adds pods that the chart's PMM_HA_PEERS doesn't list; they join through the listed ones.
	set := statefulSetName(params.NodeID)
	return set == "" || set != statefulSetName(ids[0])
}

// statefulSetName returns the StatefulSet name of a pod name such as pmm-ha-3, or "" if the name has no ordinal.
func statefulSetName(pod string) string {
	i := strings.LastIndex(pod, "-")
	if i <= 0 {
		return ""
	}
	_, err := strconv.Atoi(pod[i+1:])
	if err != nil {
		return ""
	}
	return pod[:i]
}

// New provides a new instance of the high availability service.
func New(params *models.HAParams) *Service {
	return &Service{
		params:           params,
		bootstrapCluster: bootstrapsCluster(params),
		services:         newServices(),
		nodeCh:           make(chan memberlist.NodeEvent, defaultNodeEventChanSize),
		leaderCh:         make(chan raft.Observation),
		delegate:         &memberlistDelegate{},
		raftReachable:    raftReachable,
		l:                logrus.WithField("component", "ha"),
		wg:               &sync.WaitGroup{},
		left:             make(map[string]struct{}),
		unreachable:      make(map[raft.ServerID]struct{}),
	}
}

// Run runs the high availability service.
func (s *Service) Run(ctx context.Context) error {
	s.wg.Go(func() {
		for {
			select {
			case <-s.services.Refresh():
				if s.IsLeader() {
					s.services.StartAllServices(ctx)
				}
			case <-ctx.Done():
				s.services.StopAllServices()
				return
			}
		}
	})

	if !s.params.Enabled {
		s.l.Infoln("High availability is disabled")
		s.wg.Wait()
		s.services.Wait()
		return nil
	}

	s.l.Infoln("Starting...")
	defer s.l.Infoln("Done.")

	// Create the Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(s.params.NodeID)
	raftConfig.LogOutput = newRaftLogLimiter(s.l.Logger.Out, defaultRaftLogRepeatInterval)

	// Set log level based on environment
	if os.Getenv("PMM_DEBUG") == "1" {
		raftConfig.LogLevel = "DEBUG"
	} else {
		raftConfig.LogLevel = "WARN"
	}

	// Configure timeouts for better cluster stability
	raftConfig.HeartbeatTimeout = defaultHeartbeatTimeout
	raftConfig.ElectionTimeout = defaultElectionTimeout
	raftConfig.CommitTimeout = defaultCommitTimeout
	raftConfig.LeaderLeaseTimeout = defaultLeaderLeaseTimeout

	// Configure snapshots for log compaction
	// Since PMM HA is stateless (leader election only), snapshots are minimal
	raftConfig.SnapshotInterval = defaultSnapshotInterval
	raftConfig.SnapshotThreshold = defaultSnapshotThreshold
	raftConfig.TrailingLogs = defaultTrailingLogs

	tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(s.params.AdvertiseAddress, strconv.Itoa(s.params.RaftPort)))
	if err != nil {
		return err
	}

	raftTrans, err := raft.NewTCPTransport(
		net.JoinHostPort("0.0.0.0", strconv.Itoa(s.params.RaftPort)),
		tcpAddr,
		defaultRaftRetries,
		defaultTransportTimeout,
		nil,
	)
	if err != nil {
		return err
	}

	// Set up persistent storage for Raft
	logStore, stableStore, snapshotStore, err := setupRaftStorage(s.params.NodeID, s.l)
	if err != nil {
		return err
	}

	defer func() {
		if logStore != nil {
			closeErr := logStore.Close()
			if closeErr != nil {
				s.l.Errorf("error closing log store: %v", closeErr)
			}
		}
		if stableStore != nil {
			closeErr := stableStore.Close()
			if closeErr != nil {
				s.l.Errorf("error closing stable store: %v", closeErr)
			}
		}
	}()

	// Create a new Raft node with persistent storage
	s.rw.Lock()
	s.raftNode, err = raft.NewRaft(raftConfig, s, logStore, stableStore, snapshotStore, raftTrans)
	s.rw.Unlock()
	if err != nil {
		return err
	}
	defer func() {
		if s.IsLeader() && s.hasOtherVoters() {
			err := s.raftNode.LeadershipTransfer().Error()
			if err != nil {
				s.l.Warnf("Couldn't transfer leadership: %s", err)
			}
		}
		err := s.raftNode.Shutdown().Error()
		if err != nil {
			s.l.Errorf("error during the shutdown of raft node: %q", err)
		}
	}()

	// Create the memberlist configuration
	memberlistConfig := memberlist.DefaultWANConfig()
	memberlistConfig.Name = s.params.NodeID
	memberlistConfig.BindAddr = "0.0.0.0"
	memberlistConfig.BindPort = s.params.GossipPort
	memberlistConfig.AdvertiseAddr = s.params.AdvertiseAddress
	memberlistConfig.AdvertisePort = s.params.GossipPort
	memberlistConfig.Events = &memberlist.ChannelEventDelegate{Ch: s.nodeCh}
	memberlistConfig.Delegate = s.delegate
	memberlistConfig.LogOutput = newMemberlistLogWriter(s.l.WithField("subsystem", "memberlist"))

	// Create the memberlist
	s.memberlist, err = memberlist.Create(memberlistConfig)
	if err != nil {
		return fmt.Errorf("failed to create memberlist: %w", err)
	}
	// memberlist delivers node events while holding its lock, so keep reading them from the moment shutdown starts,
	// when runRaftNodesSynchronizer stops, until memberlist has shut down.
	drained := make(chan struct{})
	drain := func() {
		for {
			select {
			case <-s.nodeCh:
			case <-drained:
				return
			}
		}
	}
	stopDrain := context.AfterFunc(ctx, drain)
	defer func() {
		// Run can also return early on an error, before the context is canceled.
		if stopDrain() {
			go drain()
		}
		s.leaveMemberlist()
		close(drained)
	}()

	s.startHeartbeatObserver(ctx)

	if s.bootstrapCluster {
		// Start the Raft node
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(s.params.NodeID),
					Address:  raft.ServerAddress(net.JoinHostPort(s.lookupFQDN(ctx, s.params.AdvertiseAddress), strconv.Itoa(s.params.RaftPort))),
				},
			},
		}
		err := s.raftNode.BootstrapCluster(cfg).Error()
		if err != nil {
			// Cluster might already be bootstrapped with persistent storage
			if !errors.Is(err, raft.ErrCantBootstrap) {
				return fmt.Errorf("failed to bootstrap Raft cluster: %w", err)
			}
			s.l.Info("Cluster already bootstrapped, skipping")
		}
	} else {
		s.l.Info("Not bootstrapping the Raft cluster: the first node in PMM_HA_PEERS does, and its leader adds this node")
	}
	if len(s.params.Nodes) != 0 {
		_, err := s.memberlist.Join(s.params.Nodes)
		if err != nil {
			return fmt.Errorf("failed to join memberlist cluster: %w", err)
		}
	}
	s.wg.Go(func() {
		s.runLeaderObserver(ctx)
	})

	s.wg.Go(func() {
		s.runRaftNodesSynchronizer(ctx)
	})

	s.wg.Go(func() {
		s.runMemberlistRejoiner(ctx)
	})

	<-ctx.Done()

	s.wg.Wait()
	s.services.Wait()

	return nil
}

// hasOtherVoters reports whether the Raft configuration has a voter besides this node to hand leadership to.
func (s *Service) hasOtherVoters() bool {
	future := s.raftNode.GetConfiguration()
	if future.Error() != nil {
		return true
	}
	return slices.ContainsFunc(future.Configuration().Servers, func(server raft.Server) bool {
		return server.Suffrage == raft.Voter && string(server.ID) != s.params.NodeID
	})
}

// leaveMemberlist announces a graceful leave to the other nodes, then leaves memberlist and shuts it down.
func (s *Service) leaveMemberlist() {
	if s.memberlist.NumMembers() > 1 {
		s.delegate.leaving.Store(true)
		err := s.memberlist.UpdateNode(defaultLeaveAnnounceTimeout)
		if err != nil {
			s.l.Warnf("Couldn't announce leaving the memberlist cluster: %s", err)
		}
	}
	err := s.memberlist.Leave(defaultLeaveTimeout)
	if err != nil {
		s.l.Errorf("couldn't leave memberlist cluster: %q", err)
	}
	err = s.memberlist.Shutdown()
	if err != nil {
		s.l.Errorf("couldn't shutdown memberlist listeners: %q", err)
	}
}

// startHeartbeatObserver tracks which voters this node, while it is the leader, fails to heartbeat over Raft.
// It catches a peer that accepts connections but doesn't answer, which memberlist only declares dead after over 30 s.
func (s *Service) startHeartbeatObserver(ctx context.Context) {
	observations := make(chan raft.Observation, defaultObservationChanSize)
	s.raftNode.RegisterObserver(raft.NewObserver(observations, false, func(o *raft.Observation) bool {
		switch o.Data.(type) {
		case raft.FailedHeartbeatObservation, raft.ResumedHeartbeatObservation, raft.LeaderObservation:
			return true
		default:
			return false
		}
	}))

	s.wg.Go(func() {
		for {
			select {
			case o := <-observations:
				switch data := o.Data.(type) {
				case raft.FailedHeartbeatObservation:
					s.setUnreachable(data.PeerID, true)
				case raft.ResumedHeartbeatObservation:
					s.setUnreachable(data.PeerID, false)
				case raft.LeaderObservation:
					s.peersMu.Lock()
					clear(s.unreachable)
					s.peersMu.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	})
}

func (s *Service) runRaftNodesSynchronizer(ctx context.Context) {
	t := time.NewTicker(defaultTickerInterval)
	defer t.Stop()

	for {
		select {
		case event := <-s.nodeCh:
			s.handleNodeEvent(ctx, event)
		case <-t.C:
			if !s.IsLeader() {
				continue
			}

			// Periodically reconcile Raft with memberlist to handle any missed events
			s.l.Debug("Running periodic Raft-memberlist reconciliation")
			s.reconcileRaftWithMemberlist(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// handleNodeEvent records graceful leaves on every node, so that a later leader knows about them too.
// On the leader it adds joined nodes to Raft and removes nodes that left; a failed node stays a voter, see PMM-15787.
func (s *Service) handleNodeEvent(ctx context.Context, event memberlist.NodeEvent) {
	node := event.Node
	switch event.Event {
	case memberlist.NodeJoin:
		s.setLeft(node.Name, false)
		if s.IsLeader() {
			s.addMemberlistNodeToRaft(ctx, node)
		}
	case memberlist.NodeLeave:
		if string(node.Meta) != memberlistMetaLeaving {
			s.l.Warnf("Node %s failed, keeping it in Raft", node.Name)
			return
		}
		s.l.Infof("Node %s left the cluster", node.Name)
		s.setLeft(node.Name, true)
		if s.IsLeader() {
			s.reconcileRaftWithMemberlist(ctx)
		}
	case memberlist.NodeUpdate:
	}
}

// aliveQuorumWithout reports whether a majority of the voters that remain after removing id is alive.
func aliveQuorumWithout(servers []raft.Server, alive map[string]struct{}, id raft.ServerID) bool {
	var voters, aliveVoters int
	for _, server := range servers {
		if server.ID == id || server.Suffrage != raft.Voter {
			continue
		}
		voters++
		if _, ok := alive[string(server.ID)]; ok {
			aliveVoters++
		}
	}
	return aliveVoters*2 > voters
}

func (s *Service) setLeft(name string, left bool) {
	s.peersMu.Lock()
	defer s.peersMu.Unlock()
	if left {
		s.left[name] = struct{}{}
	} else {
		delete(s.left, name)
	}
}

func (s *Service) hasLeft(name string) bool {
	s.peersMu.Lock()
	defer s.peersMu.Unlock()
	_, ok := s.left[name]
	return ok
}

// liveVoters returns the voters that are alive in memberlist, aren't failing Raft heartbeats and accept
// a connection on their Raft address: memberlist and Raft can take 10 to 30 s to notice a lost node.
func (s *Service) liveVoters(ctx context.Context, servers []raft.Server, members map[string]*memberlist.Node) map[string]struct{} {
	live := make(map[string]struct{})
	for _, server := range servers {
		id := string(server.ID)
		if _, ok := members[id]; !ok || server.Suffrage != raft.Voter || s.isUnreachable(server.ID) {
			continue
		}
		if id == s.params.NodeID || s.raftReachable(ctx, server.Address) {
			live[id] = struct{}{}
		}
	}
	return live
}

// raftReachable reports whether a Raft address accepts a TCP connection.
func raftReachable(ctx context.Context, address raft.ServerAddress) bool {
	dialer := net.Dialer{Timeout: defaultRaftProbeTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", string(address))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (s *Service) setUnreachable(id raft.ServerID, unreachable bool) {
	s.peersMu.Lock()
	defer s.peersMu.Unlock()
	if unreachable {
		s.unreachable[id] = struct{}{}
	} else {
		delete(s.unreachable, id)
	}
}

func (s *Service) isUnreachable(id raft.ServerID) bool {
	s.peersMu.Lock()
	defer s.peersMu.Unlock()
	_, ok := s.unreachable[id]
	return ok
}

// runMemberlistRejoiner periodically joins the configured peers again while some of them are missing from memberlist.
func (s *Service) runMemberlistRejoiner(ctx context.Context) {
	t := time.NewTicker(defaultRejoinInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			s.rejoinMemberlist(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// rejoinMemberlist joins the reachable peers when memberlist lists fewer nodes than configured,
// since memberlist stops contacting a node it has marked dead and a healed partition would stay split.
func (s *Service) rejoinMemberlist(ctx context.Context) {
	if s.memberlist.NumMembers() >= s.expectedNodes() {
		return
	}

	dialer := net.Dialer{Timeout: defaultRejoinDialTimeout}
	for _, peer := range s.params.Nodes {
		address := peer
		_, _, err := net.SplitHostPort(peer)
		if err != nil {
			address = net.JoinHostPort(peer, strconv.Itoa(s.params.GossipPort))
		}

		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			s.l.Debugf("Memberlist peer %s is not reachable: %v", address, err)
			continue
		}
		err = conn.Close()
		if err != nil {
			s.l.Debugf("Couldn't close the probe connection to %s: %v", address, err)
		}

		// memberlist.Join takes no context and can wait out its 30 s TCP timeout on a peer that accepts
		// connections but doesn't answer, so it must not hold up shutdown.
		joined := make(chan error, 1)
		go func() {
			_, err := s.memberlist.Join([]string{address})
			joined <- err
		}()
		select {
		case err = <-joined:
			if err != nil {
				s.l.Warnf("Couldn't rejoin memberlist peer %s: %v", address, err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// reconcileRaftWithMemberlist ensures Raft cluster configuration matches memberlist membership.
// This method is invoked when a node becomes the Raft leader. It reconciles
// any missed NodeJoin/NodeLeave events and removes Raft members that left the cluster gracefully.
func (s *Service) reconcileRaftWithMemberlist(ctx context.Context) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	// Fetch the current Raft cluster configuration.
	configFuture := s.raftNode.GetConfiguration()
	err := configFuture.Error()
	if err != nil {
		s.l.Errorf("failed to get raft configuration for reconciliation: %v", err)
		return
	}

	raftServers := configFuture.Configuration().Servers
	members := s.memberlist.Members()

	// Build a map of current memberlist members for quick lookup
	memberMap := make(map[string]*memberlist.Node)
	for _, member := range members {
		memberMap[member.Name] = member
	}

	// Build a map of current Raft servers
	raftServerMap := make(map[string]struct{})
	for _, server := range raftServers {
		raftServerMap[string(server.ID)] = struct{}{}
	}

	// Remove nodes that are in Raft but left the cluster gracefully. A failed node stays a voter,
	// so that it can vote again when it returns, see PMM-15787.
	servers := raftServers
	for _, server := range raftServers {
		serverID := string(server.ID)
		if _, exists := memberMap[serverID]; exists || !s.hasLeft(serverID) {
			continue
		}
		// Raft counts quorum on the new configuration as soon as the change is appended, so a removal
		// that the remaining live voters can't commit would make this leader step down.
		if !aliveQuorumWithout(servers, s.liveVoters(ctx, servers, memberMap), server.ID) {
			s.l.Debugf("Keeping node %s in Raft: too few voters are alive to commit its removal", serverID)
			continue
		}
		s.l.Warnf("Removing node %s from Raft (left the cluster)", serverID)
		err := s.raftNode.RemoveServer(server.ID, 0, defaultServerOpTimeout).Error()
		if err != nil {
			s.l.Errorf("Failed to remove server %s from Raft: %v", serverID, err)
			continue
		}
		s.l.Infof("Successfully removed node %s from Raft cluster", serverID)
		servers = slices.DeleteFunc(slices.Clone(servers), func(srv raft.Server) bool { return srv.ID == server.ID })
	}

	// Add nodes that are in memberlist but NOT in Raft
	for _, member := range members {
		if member.Name == s.params.NodeID {
			continue
		}
		if _, exists := raftServerMap[member.Name]; !exists {
			hostname := s.lookupFQDN(ctx, member.Addr.String())
			serverAddress := raft.ServerAddress(fmt.Sprintf("%s:%d", hostname, s.params.RaftPort))
			s.l.Infof("Adding missing node %s to Raft (in memberlist but not in Raft)", member.Name)
			err := s.raftNode.AddVoter(raft.ServerID(member.Name), serverAddress, 0, defaultServerOpTimeout).Error()
			if err != nil {
				s.l.Errorf("Failed to add server %s to Raft: %v", member.Name, err)
			} else {
				s.l.Infof("Successfully added node %s to Raft cluster with address: %s", member.Name, serverAddress)
				s.setUnreachable(raft.ServerID(member.Name), false)
			}
		}
	}
}

func (s *Service) addMemberlistNodeToRaft(ctx context.Context, node *memberlist.Node) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	hostname := s.lookupFQDN(ctx, node.Addr.String())
	serverAddress := raft.ServerAddress(fmt.Sprintf("%s:%d", hostname, s.params.RaftPort))

	err := s.raftNode.AddVoter(raft.ServerID(node.Name), serverAddress, 0, defaultServerOpTimeout).Error()
	if err != nil {
		s.l.Errorf("Couldn't add a server node %s (address: %s): %s", node.Name, serverAddress, err)
	} else {
		s.l.Infof("Added node %s to Raft cluster with address: %s", node.Name, serverAddress)
		s.setUnreachable(raft.ServerID(node.Name), false)
	}
}

// lookupFQDN performs reverse DNS lookup to get FQDN from IP address.
func (s *Service) lookupFQDN(ctx context.Context, address string) string {
	if net.ParseIP(address) == nil {
		return address
	}

	lookupCtx, cancel := context.WithTimeout(ctx, defaultDNSLookupTimeout)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(lookupCtx, address)
	if err != nil || len(names) == 0 {
		s.l.Warnf("Failed to lookup FQDN for %s, using IP: %s", address, err)
		return address
	}

	fqdn := strings.TrimSuffix(names[0], ".")
	s.l.Debugf("Resolved %s to FQDN: %s", address, fqdn)
	return fqdn
}

func (s *Service) runLeaderObserver(ctx context.Context) {
	t := time.NewTicker(defaultTickerInterval)
	defer t.Stop()

	for {
		s.rw.RLock()
		node := s.raftNode
		s.rw.RUnlock()
		select {
		case isLeader := <-node.LeaderCh():
			if isLeader {
				s.services.StartAllServices(ctx)
				s.l.Info("I am the leader!")

				// Reconcile Raft configuration with memberlist
				s.reconcileRaftWithMemberlist(ctx)
			} else {
				s.l.Info("I am not a leader!")
				s.services.StopAllServices()
			}
		case <-t.C:
			address, serverID := node.LeaderWithID()
			if serverID != "" {
				s.l.Debugf("Leader is %s on %s", serverID, address)
			}
		case <-ctx.Done():
			return
		}
	}
}

// AddLeaderService adds a leader service to the high availability service.
func (s *Service) AddLeaderService(leaderService LeaderService) {
	err := s.services.Add(leaderService)
	if err != nil {
		s.l.Errorf("couldn't add HA service: %+v", err)
	}
}

// BroadcastMessage broadcasts a message from the high availability service.
// Note: Currently unused. Reserved for future cluster-wide message distribution.
// This method should only be called by the leader node.
func (s *Service) BroadcastMessage(message []byte) error {
	if !s.params.Enabled {
		return errors.New("HA is disabled")
	}

	s.rw.RLock()
	defer s.rw.RUnlock()

	future := s.raftNode.Apply(message, defaultApplyTimeout)

	err := future.Error()
	if err != nil {
		return fmt.Errorf("failed to apply log to raft: %w", err)
	}
	return nil
}

// IsLeader checks if the current instance of HA service is the leader.
func (s *Service) IsLeader() bool {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return !s.params.Enabled || (s.raftNode != nil && s.raftNode.State() == raft.Leader)
}

// LeaderID returns the current Raft leader's node ID, or "" when HA is disabled or no leader is known.
func (s *Service) LeaderID() string {
	s.rw.RLock()
	defer s.rw.RUnlock()
	if !s.params.Enabled || s.raftNode == nil {
		return ""
	}
	_, leaderID := s.raftNode.LeaderWithID()
	return string(leaderID)
}

// Params returns HA parameters.
func (s *Service) Params() *models.HAParams {
	return s.params
}

// expectedNodes returns the number of PMM Server nodes declared for this cluster.
// Defaults to 1 for a single-node deployment where no peers are configured.
func (s *Service) expectedNodes() int {
	return max(len(s.params.Nodes), 1)
}

// Metrics holds HA-related Prometheus metric values for this node.
type Metrics struct {
	// Enabled indicates whether HA mode is active.
	Enabled bool
	// IsLeader is true when this node currently holds the Raft leader lease.
	IsLeader bool
	// RaftTerm is the current Raft consensus term. Rapid increases indicate
	// an unstable leader or frequent elections (leader flapping).
	RaftTerm uint64
	// IsVoter is true when this node is in the current Raft configuration and votes
	// in leader elections. It is false when the node is absent from that configuration,
	// and also when the configuration could not be read. PMM never adds true Raft
	// non-voting members, so a false value always means "not in the configuration".
	IsVoter bool
	// ExpectedNodes is the number of nodes configured for this cluster.
	ExpectedNodes int
}

// GetMetrics returns current HA Raft metrics for this node. The returned
// values are intended to be exposed as Prometheus gauges so that VictoriaMetrics
// can evaluate the built-in cluster-health alert templates: pmm_ha_no_leader,
// pmm_ha_split_brain, pmm_ha_leader_flapping, pmm_ha_node_unreachable and
// pmm_ha_quorum_at_risk.
//
// When HA is disabled, Enabled is false and all other fields are zero values.
func (s *Service) GetMetrics() Metrics {
	if !s.params.Enabled {
		return Metrics{Enabled: false}
	}

	expectedNodes := s.expectedNodes()

	s.rw.RLock()
	raftNode := s.raftNode
	s.rw.RUnlock()

	if raftNode == nil {
		// HA enabled but Raft not yet initialised (early startup).
		return Metrics{Enabled: true, ExpectedNodes: expectedNodes}
	}

	isLeader := raftNode.State() == raft.Leader

	// Extract the current Raft term from the stats map (returned as a decimal string).
	var term uint64
	stats := raftNode.Stats()
	if termStr, ok := stats["term"]; ok {
		term, _ = strconv.ParseUint(termStr, 10, 64)
	}

	// Determine whether this node is configured as a Raft voter.
	isVoter := false
	configFuture := raftNode.GetConfiguration()
	err := configFuture.Error()
	if err == nil {
		for _, server := range configFuture.Configuration().Servers {
			if server.ID == raft.ServerID(s.params.NodeID) {
				isVoter = server.Suffrage == raft.Voter
				break
			}
		}
	}

	return Metrics{
		Enabled:       true,
		IsLeader:      isLeader,
		RaftTerm:      term,
		IsVoter:       isVoter,
		ExpectedNodes: expectedNodes,
	}
}
