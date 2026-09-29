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

package models

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkUniqueNodeID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Node ID")
	}

	node := &Node{NodeID: id}
	err := q.Reload(node)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return err
	}

	return status.Errorf(codes.AlreadyExists, "Node with ID %q already exists.", id)
}

func checkUniqueNodeName(q *reform.Querier, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	_, err := q.FindOneFrom(NodeTable, "node_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return err
	}

	return status.Errorf(codes.AlreadyExists, "Node with name %s already exists.", name)
}

// CheckUniqueNodeAddressRegion checks for uniqueness of instance address and region.
// This function not only returns an error in case it finds an existing node with the same address and region, but
// also returns the Node itself if there is any, because if we are recreating the instance (--force in pmm-admin)
// we need to know the Node.ID to remove it along with its dependencies.
// This check is performed only if the region is not empty.
func CheckUniqueNodeAddressRegion(q *reform.Querier, address string, region *string) (*Node, error) {
	if pointer.GetString(region) == "" {
		return nil, nil //nolint:nilnil
	}

	if address == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node address.")
	}

	var node Node
	err := q.SelectOneTo(&node, "WHERE address = $1 AND region = $2 LIMIT 1", address, region)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil //nolint:nilnil
		}
		return nil, err
	}

	return &node, status.Errorf(codes.AlreadyExists, "Node with address %q and region %q already exists.", address, *region)
}

// NodeFilters represents filters for nodes list.
type NodeFilters struct {
	// Return Nodes with provided type.
	NodeType *NodeType
	// Return only Nodes that are (or are not) PMM Server Nodes.
	IsPMMServerNode *bool
}

// FindNodes returns Nodes by filters.
func FindNodes(q *reform.Querier, filters NodeFilters) ([]*Node, error) {
	var conditions []string
	var args []any
	idx := 1
	if filters.NodeType != nil {
		conditions = append(conditions, "node_type = "+q.Placeholder(idx))
		args = append(args, *filters.NodeType)
		idx++
	}
	if filters.IsPMMServerNode != nil {
		conditions = append(conditions, "is_pmm_server_node = "+q.Placeholder(idx))
		args = append(args, *filters.IsPMMServerNode)
		// idx++
	}
	var whereClause string
	if len(conditions) != 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	structs, err := q.SelectAllFrom(NodeTable, whereClause+" ORDER BY node_id", args...)
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, len(structs))
	for i, s := range structs {
		nodes[i] = s.(*Node) //nolint:forcetypeassert
	}

	return nodes, nil
}

// FindNodeByID finds a Node by ID.
func FindNodeByID(q *reform.Querier, id string) (*Node, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node ID.")
	}

	node := &Node{NodeID: id}
	err := q.Reload(node)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Node with ID %q not found.", id)
		}
		return nil, err
	}
	return node, nil
}

// FindNodesByIDs finds Nodes by IDs.
func FindNodesByIDs(q *reform.Querier, ids []string) ([]*Node, error) {
	if len(ids) == 0 {
		return []*Node{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE node_id IN (%s) ORDER BY node_id", p)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	structs, err := q.SelectAllFrom(NodeTable, tail, args...)
	if err != nil {
		return nil, err
	}

	res := make([]*Node, len(structs))
	for i, s := range structs {
		res[i] = s.(*Node) //nolint:forcetypeassert
	}
	return res, nil
}

// FindNodeByName finds a Node by name.
func FindNodeByName(q *reform.Querier, name string) (*Node, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	var node Node
	err := q.FindOneTo(&node, "node_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Node with name %q not found.", name)
		}
		return nil, err
	}

	return &node, nil
}

// CreateNodeParams contains parameters for creating Nodes.
type CreateNodeParams struct {
	NodeName        string
	MachineID       *string
	Distro          string
	NodeModel       string
	AZ              string
	ContainerID     *string
	ContainerName   *string
	CustomLabels    map[string]string
	Address         string
	InstanceID      string
	Region          *string
	Password        *string
	IsPMMServerNode bool
}

// createNodeWithID creates a Node with given ID.
func createNodeWithID(q *reform.Querier, id string, nodeType NodeType, params *CreateNodeParams) (*Node, error) {
	err := checkUniqueNodeID(q, id)
	if err != nil {
		return nil, err
	}

	err = checkUniqueNodeName(q, params.NodeName)
	if err != nil {
		return nil, err
	}

	// do not check that machine-id is unique: https://perconadev.atlassian.net/browse/PMM-4196

	if nodeType == RemoteRDSNodeType {
		if strings.Contains(params.InstanceID, ".") {
			return nil, status.Error(codes.InvalidArgument, "DB instance identifier should not contain dots.")
		}
	}

	_, err = CheckUniqueNodeAddressRegion(q, params.Address, params.Region)
	if err != nil {
		return nil, err
	}

	// Trim trailing \n received from broken 2.0.0 clients.
	// See https://perconadev.atlassian.net/browse/PMM-4720
	machineID := pointer.ToStringOrNil(strings.TrimSpace(pointer.GetString(params.MachineID)))

	node := &Node{
		NodeID:          id,
		NodeType:        nodeType,
		NodeName:        params.NodeName,
		MachineID:       machineID,
		Distro:          params.Distro,
		NodeModel:       params.NodeModel,
		AZ:              params.AZ,
		ContainerID:     params.ContainerID,
		ContainerName:   params.ContainerName,
		InstanceID:      params.InstanceID,
		Address:         params.Address,
		Region:          params.Region,
		IsPMMServerNode: params.IsPMMServerNode,
	}
	err = node.SetCustomLabels(params.CustomLabels)
	if err != nil {
		return nil, err
	}
	err = q.Insert(node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// CreateNode creates a Node.
func CreateNode(q *reform.Querier, nodeType NodeType, params *CreateNodeParams) (*Node, error) {
	id := uuid.New().String()
	return createNodeWithID(q, id, nodeType, params)
}

// RemoveNode removes single Node.
func RemoveNode(q *reform.Querier, id string, mode RemoveMode) error {
	return removeNode(q, id, mode, false)
}

// removeNode removes a single Node. The allowPMMServerNode flag lifts the ban on Nodes flagged as PMM
// Server Nodes; only the HA cleanup sets it, to reap replicas that are no longer part of the cluster.
func removeNode(q *reform.Querier, id string, mode RemoveMode, allowPMMServerNode bool) error { //nolint:gocognit
	n, err := FindNodeByID(q, id)
	if err != nil {
		return err
	}

	if id == defaultPMMServerNodeID || (!allowPMMServerNode && n.IsPMMServerNode) {
		return status.Error(codes.PermissionDenied, "PMM Server node can't be removed.")
	}

	// check/remove Agents
	structs, err := q.FindAllFrom(AgentTable, "node_id", id)
	if err != nil {
		return fmt.Errorf("failed to select Agent IDs: %w", err)
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Node with ID %q has agents.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID //nolint:forcetypeassert
				_, err = RemoveAgent(q, agentID, RemoveCascade)
				if err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	// check/remove pmm-agents
	structs, err = q.FindAllFrom(AgentTable, "runs_on_node_id", id)
	if err != nil {
		return fmt.Errorf("failed to select Agents: %w", err)
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Node with ID %q has pmm-agent.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID //nolint:forcetypeassert
				_, err = RemoveAgent(q, agentID, RemoveCascade)
				if err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	// check/remove Services
	structs, err = q.FindAllFrom(ServiceTable, "node_id", id)
	if err != nil {
		return fmt.Errorf("failed to select Service IDs: %w", err)
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Node with ID %q has services.", id)
		case RemoveCascade:
			for _, str := range structs {
				serviceID := str.(*Service).ServiceID //nolint:forcetypeassert
				err = RemoveService(q, serviceID, RemoveCascade)
				if err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	err = q.Delete(n)
	if err != nil {
		return fmt.Errorf("failed to delete Node: %w", err)
	}
	return nil
}

// FindStaleHANodes returns the PMM Server Nodes of HA replicas that are no longer configured peers,
// e.g. after a scale-down, and that can be removed without taking a user's monitoring with them.
// Peers are the source of truth because they are regenerated from the replica count and restart
// every replica, while a missing memberlist member may just be restarting.
//
// The localHANodeID argument is this replica's own PMM_HA_NODE_ID, which is also the Node name it
// registers under. A peer list of bare addresses names nobody and yields no Nodes; one that names no
// peers at all, or omits this replica, is an error rather than an empty result.
func FindStaleHANodes(q *reform.Querier, localHANodeID string, haPeers []string) ([]*Node, error) {
	l := logrus.WithFields(logrus.Fields{"component": "ha", "ha_node_id": localHANodeID})

	expected := make(map[string]struct{}, len(haPeers))
	for _, peer := range haPeers {
		// A trailing comma in PMM_HA_PEERS, or a blank element in the list the chart joins, yields an
		// empty entry. It names no replica, so unlike an unreadable one it hides nothing.
		if strings.TrimSpace(peer) == "" {
			continue
		}

		name, ok := haPeerNodeName(peer)
		if !ok {
			// Trusting the rest would treat a partial list as the whole cluster and remove live replicas.
			l.WithField("peer", peer).Warn("Can't read a node name from a PMM_HA_PEERS entry, so no stale HA nodes are reported.")
			return nil, nil
		}
		expected[name] = struct{}{}
	}

	// The chart lists every replica including the pod reading it, down to replicas=1, so neither
	// case describes a cluster this replica belongs to.
	if len(expected) == 0 {
		return nil, errors.New("PMM_HA_PEERS names no peers")
	}
	if _, ok := expected[localHANodeID]; !ok {
		return nil, fmt.Errorf("PMM_HA_PEERS (%s) doesn't list this node", strings.Join(haPeers, ","))
	}

	// Only PMM Server Nodes can be stale replicas; the rest are Nodes the user monitors.
	nodes, err := FindNodes(q, NodeFilters{IsPMMServerNode: new(true)})
	if err != nil {
		return nil, fmt.Errorf("failed to list Nodes for stale HA node cleanup: %w", err)
	}

	var stale []*Node
	for _, node := range nodes {
		// The PMM Server Node of a deployment converted from non-HA: HA replicas always get a
		// generated Node ID, and removeNode bans this one outright. Compared against the const,
		// not PMMServerNodeID: setupPMMServerHAAgents reassigns that var, and SetupDB is retried.
		if node.NodeID == defaultPMMServerNodeID {
			continue
		}
		if _, ok := expected[node.NodeName]; ok {
			continue
		}

		nodeL := l.WithFields(logrus.Fields{"node_id": node.NodeID, "node_name": node.NodeName})

		// Keeping a Node is per Node: another replica removing the same rows concurrently must not
		// hide the Nodes this pass hasn't reached yet.
		monitored, err := haNodeMonitoredServices(q, node.NodeID)
		if err != nil {
			nodeL.WithError(err).Warn("Can't tell whether a stale HA node monitors services, keeping it.")
			continue
		}
		if len(monitored) != 0 {
			nodeL.WithField("service_ids", monitored).Warn("Keeping stale HA node: it still monitors services, which would be removed with it. " +
				"Re-add them from a running replica; the next restart removes the node.")
			continue
		}

		stale = append(stale, node)
	}

	return stale, nil
}

// RemoveStaleHANode removes a Node returned by FindStaleHANodes together with its Agents. Call it in a
// transaction of its own: it deletes those before the Node itself, so a failure half-way through
// would otherwise leave the Node partially removed.
//
// It lifts the ban on removing PMM Server Nodes, so it re-reads the Services the Node monitors and
// refuses to take a Node that has any.
func RemoveStaleHANode(q *reform.Querier, nodeID string) error {
	monitored, err := haNodeMonitoredServices(q, nodeID)
	if err != nil {
		return err
	}
	if len(monitored) != 0 {
		return status.Errorf(codes.FailedPrecondition, "HA Node with ID %q still monitors services.", nodeID)
	}

	return removeNode(q, nodeID, RemoveCascade, true)
}

// haPeerNodeName maps a PMM_HA_PEERS entry ("pmm-ha-0.pmm-ha.pmm.svc.cluster.local:9761") to a Node
// name: the first label is the pod's PMM_HA_NODE_ID. Reports false for entries with no name, like
// bare IPv4 or IPv6 addresses.
func haPeerNodeName(peer string) (string, bool) {
	peer = strings.TrimSpace(peer)
	// Test the whole entry before cutting at ":": an unbracketed IPv6 literal would otherwise be cut
	// into its first group, and the "2001" of "2001:db8::7" reads like a node name. Only IPv6 entries
	// hold more than one colon, bracketed or not, and none of them starts with a name.
	if strings.Count(peer, ":") > 1 || net.ParseIP(peer) != nil {
		return "", false
	}
	host, _, _ := strings.Cut(peer, ":")
	if net.ParseIP(host) != nil {
		return "", false
	}
	// "/" is memberlist's "name/address" form, "[" a bracketed address; such a label mixes a name
	// with an address instead of being one.
	label, _, _ := strings.Cut(host, ".")
	if label == "" || strings.ContainsAny(label, "/[") {
		return "", false
	}
	return label, true
}

// haNodeMonitoredServices returns the IDs of the Services that removing the Node would damage: those
// attached to the Node, which removeNode deletes outright, and those whose exporters run on it (an
// external exporter in pull mode) or under its pmm-agent (remote instances bind theirs to the replica
// that added them), which survive but lose their monitoring. A Node with any of them is kept.
func haNodeMonitoredServices(q *reform.Querier, nodeID string) ([]string, error) {
	// An external exporter carries a service_id itself, and the pmm-agents are the parents of the
	// exporters read next.
	agents, err := FindAgents(q, AgentFilters{OnNodeID: nodeID})
	if err != nil {
		return nil, err
	}

	var serviceIDs, pmmAgentIDs []string
	for _, agent := range agents {
		if agent.ServiceID != nil {
			serviceIDs = append(serviceIDs, *agent.ServiceID)
		}
		if agent.AgentType == PMMAgentType {
			pmmAgentIDs = append(pmmAgentIDs, agent.AgentID)
		}
	}

	// Guarded because an empty PMMAgentIDs is not a filter: it would match every Agent in the
	// inventory, and the Node would look like it monitors every Service.
	if len(pmmAgentIDs) != 0 {
		started, err := FindAgents(q, AgentFilters{PMMAgentIDs: pmmAgentIDs})
		if err != nil {
			return nil, err
		}
		for _, agent := range started {
			if agent.ServiceID != nil {
				serviceIDs = append(serviceIDs, *agent.ServiceID)
			}
		}
	}

	services, err := FindServices(q, ServiceFilters{NodeID: nodeID})
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		serviceIDs = append(serviceIDs, service.ServiceID)
	}

	return deduplicateStrings(serviceIDs), nil
}
