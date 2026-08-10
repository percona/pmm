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
}

// FindNodes returns Nodes by filters.
func FindNodes(q *reform.Querier, filters NodeFilters) ([]*Node, error) {
	var whereClause string
	var args []any
	if filters.NodeType != nil {
		whereClause = "WHERE node_type = $1"
		args = append(args, *filters.NodeType)
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

	if id == PMMServerNodeID || (!allowPMMServerNode && n.IsPMMServerNode) {
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

// RemoveStaleHANodes removes the PMM Server Nodes of HA replicas that are no longer configured peers,
// e.g. after a scale-down. Peers are the source of truth because they are regenerated from the replica
// count and restart every replica, while a missing memberlist member may just be restarting.
func RemoveStaleHANodes(q *reform.Querier, haNodeID string, haPeers []string) error {
	if len(haPeers) == 0 {
		return nil
	}

	l := logrus.WithFields(logrus.Fields{"component": "ha", "ha_node_id": haNodeID})

	expected := make(map[string]struct{}, len(haPeers))
	for _, peer := range haPeers {
		name, ok := haPeerNodeName(peer)
		if !ok {
			// Trusting the rest would treat a partial list as the whole cluster and remove live replicas.
			l.WithField("peer", peer).Warn("Can't read a node name from a PMM_HA_PEERS entry, skipping the removal of stale HA nodes.")
			return nil
		}
		expected[name] = struct{}{}
	}

	if _, ok := expected[haNodeID]; !ok {
		l.WithField("ha_peers", haPeers).Warn("PMM_HA_PEERS doesn't list this node, skipping the removal of stale HA nodes.")
		return nil
	}

	nodes, err := FindNodes(q, NodeFilters{})
	if err != nil {
		return fmt.Errorf("failed to list Nodes for stale HA node cleanup: %w", err)
	}

	for _, node := range nodes {
		// Set by HA replicas, and by the PMM Server Node of a non-HA deployment; every other
		// Node is one the user monitors.
		if !node.IsPMMServerNode {
			continue
		}
		// The PMM Server Node of a deployment converted from non-HA: HA replicas always get a
		// generated Node ID, and removeNode bans this one outright.
		if node.NodeID == PMMServerNodeID {
			continue
		}
		if _, ok := expected[node.NodeName]; ok {
			continue
		}

		nodeL := l.WithFields(logrus.Fields{"node_id": node.NodeID, "node_name": node.NodeName})

		monitored, err := haNodeMonitoredServices(q, node.NodeID)
		if err != nil {
			return err
		}
		if len(monitored) != 0 {
			nodeL.WithField("service_ids", monitored).Warn("Keeping stale HA node: it still monitors services, which would be removed with it. " +
				"Re-add them from a running replica and remove the node from Inventory.")
			continue
		}

		err = removeNode(q, node.NodeID, RemoveCascade, true)
		switch {
		case err == nil:
			nodeL.Info("Removed stale HA node, it is not a part of the cluster anymore.")
		case errors.Is(err, reform.ErrNoRows), status.Code(err) == codes.NotFound:
			nodeL.Info("Stale HA node was already removed by another replica.")
		default:
			return fmt.Errorf("failed to remove stale HA node %q: %w", node.NodeName, err)
		}
	}

	return nil
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

// haNodeMonitoredServices returns the IDs of Services whose exporters run under a replica's pmm-agent.
// Remote instances bind theirs to the replica that added them (see management.RDSService), so removing
// that replica's Node takes them with it.
func haNodeMonitoredServices(q *reform.Querier, nodeID string) ([]string, error) {
	pmmAgents, err := FindPMMAgentsRunningOnNode(q, nodeID)
	if err != nil {
		return nil, err
	}

	var serviceIDs []string
	for _, pmmAgent := range pmmAgents {
		agents, err := FindAgents(q, AgentFilters{PMMAgentID: pmmAgent.AgentID})
		if err != nil {
			return nil, err
		}
		for _, agent := range agents {
			if agent.ServiceID != nil {
				serviceIDs = append(serviceIDs, *agent.ServiceID)
			}
		}
	}

	return serviceIDs, nil
}
