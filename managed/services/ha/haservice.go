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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

const (
	defaultNodeEventChanSize  = 5
	defaultRaftRetries        = 3
	defaultTransportTimeout   = 10 * time.Second
	defaultLeaveTimeout       = 5 * time.Second
	defaultTickerInterval     = 5 * time.Second
	defaultApplyTimeout       = 3 * time.Second
	defaultRaftDataDir        = "/srv/ha"
	defaultRaftDataDirPerm    = 0o750
	defaultSnapshotRetention  = 3
	defaultSnapshotThreshold  = 8192
	defaultTrailingLogs       = 10240
	defaultHeartbeatTimeout   = 1000 * time.Millisecond
	defaultElectionTimeout    = 1000 * time.Millisecond
	defaultCommitTimeout      = 50 * time.Millisecond
	defaultLeaderLeaseTimeout = 500 * time.Millisecond
	defaultSnapshotInterval   = 120 * time.Second
	defaultServerOpTimeout    = 10 * time.Second
	defaultDNSLookupTimeout   = 3 * time.Second
)

// Service represents the high-availability service.
type Service struct {
	params           *models.HAParams
	bootstrapCluster bool

	services *services

	nodeCh   chan memberlist.NodeEvent
	leaderCh chan raft.Observation

	l  *logrus.Entry
	wg *sync.WaitGroup

	rw         sync.RWMutex
	raftNode   *raft.Raft
	memberlist *memberlist.Memberlist
}

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

// New provides a new instance of the high availability service.
func New(params *models.HAParams) *Service {
	return &Service{
		params:           params,
		bootstrapCluster: true,
		services:         newServices(),
		nodeCh:           make(chan memberlist.NodeEvent, defaultNodeEventChanSize),
		leaderCh:         make(chan raft.Observation),
		l:                logrus.WithField("component", "ha"),
		wg:               &sync.WaitGroup{},
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
	raftConfig.LogOutput = s.l.Logger.Out

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
		if s.IsLeader() {
			s.raftNode.LeadershipTransfer()
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
	memberlistConfig.LogOutput = newMemberlistLogWriter(s.l.WithField("subsystem", "memberlist"))

	// Create the memberlist
	s.memberlist, err = memberlist.Create(memberlistConfig)
	if err != nil {
		return fmt.Errorf("failed to create memberlist: %w", err)
	}
	defer func() {
		err := s.memberlist.Leave(defaultLeaveTimeout)
		if err != nil {
			s.l.Errorf("couldn't leave memberlist cluster: %q", err)
		}
		err = s.memberlist.Shutdown()
		if err != nil {
			s.l.Errorf("couldn't shutdown memberlist listeners: %q", err)
		}
	}()

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

	<-ctx.Done()

	s.wg.Wait()
	s.services.Wait()

	return nil
}

func (s *Service) runRaftNodesSynchronizer(ctx context.Context) {
	t := time.NewTicker(defaultTickerInterval)
	defer t.Stop()

	for {
		select {
		case event := <-s.nodeCh:
			if !s.IsLeader() {
				continue
			}
			node := event.Node
			switch event.Event {
			case memberlist.NodeJoin:
				s.addMemberlistNodeToRaft(ctx, node)
			case memberlist.NodeLeave:
				s.removeMemberlistNodeFromRaft(node)
			case memberlist.NodeUpdate:
				continue
			}
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

func (s *Service) removeMemberlistNodeFromRaft(node *memberlist.Node) {
	s.rw.RLock()
	defer s.rw.RUnlock()
	err := s.raftNode.RemoveServer(raft.ServerID(node.Name), 0, defaultServerOpTimeout).Error()
	if err != nil {
		s.l.Errorln(err)
	}
}

// reconcileRaftWithMemberlist ensures Raft cluster configuration matches memberlist membership.
// This method is invoked when a node becomes the Raft leader. It reconciles
// any missed NodeJoin/NodeLeave events and removes stale Raft members that
// may have accumulated while no leader was available.
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

	// Remove nodes that are in Raft but NOT in memberlist (stale nodes)
	for _, server := range raftServers {
		serverID := string(server.ID)
		if _, exists := memberMap[serverID]; !exists {
			s.l.Warnf("Removing stale node %s from Raft (not in memberlist)", serverID)
			err := s.raftNode.RemoveServer(server.ID, 0, defaultServerOpTimeout).Error()
			if err != nil {
				s.l.Errorf("Failed to remove stale server %s from Raft: %v", serverID, err)
			} else {
				s.l.Infof("Successfully removed stale node %s from Raft cluster", serverID)
			}
		}
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

// Params returns HA parameters.
func (s *Service) Params() *models.HAParams {
	return s.params
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
	// IsVoter is true when this node participates in Raft leader elections.
	// Nonvoter nodes replicate logs but never vote.
	IsVoter bool
}

// GetMetrics returns current HA Raft metrics for this node. The returned
// values are intended to be exposed as Prometheus gauges so that VictoriaMetrics
// can evaluate cluster-health alerting rules such as PMMHALeaderMissing,
// PMMHASplitBrain, PMMHALeaderFlapping and PMMHAQuorumAtRisk.
//
// When HA is disabled, Enabled is false and all other fields are zero values.
func (s *Service) GetMetrics() Metrics {
	if !s.params.Enabled {
		return Metrics{Enabled: false}
	}

	s.rw.RLock()
	raftNode := s.raftNode
	s.rw.RUnlock()

	if raftNode == nil {
		// HA enabled but Raft not yet initialised (early startup).
		return Metrics{Enabled: true}
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
		Enabled:  true,
		IsLeader: isLeader,
		RaftTerm: term,
		IsVoter:  isVoter,
	}
}
