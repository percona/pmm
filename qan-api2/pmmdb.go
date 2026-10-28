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

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

const (
	unknownTableErrorCode         = 60
	replicaAlreadyExistsErrorCode = 253
	pmmRepairInterval             = 10 * time.Minute
	leaderCheckTimeout            = 5 * time.Second
	haNodesURL                    = "http://127.0.0.1:7772/v1/ha/nodes"
	replicatedEngine              = "ENGINE = Replicated('/clickhouse/databases/{uuid}', '{shard}', '{replica}')"
)

// chNode is one ClickHouse replica of a PMM HA cluster.
type chNode struct {
	host string
	db   *sqlx.DB
}

// pmmCopy is one node's pmm database: its UUID ("" when the node has none) and metrics row count.
type pmmCopy struct {
	uuid string
	rows uint64
}

// openNodes opens a connection pool per host in the comma-separated list, with dsn's credentials.
func openNodes(dsn, list string) ([]chNode, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	var nodes []chNode
	for host := range strings.SplitSeq(list, ",") {
		if host == "" {
			continue
		}
		u.Host = net.JoinHostPort(host, u.Port())
		u.Path = "/default"
		db, err := sqlx.Open("clickhouse", u.String())
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, chNode{host: host, db: db})
	}
	return nodes, nil
}

// keptUUID returns the UUID of the copy with the most rows (the lowest UUID on a tie), or "" when no node has pmm.
func keptUUID(copies []pmmCopy) string {
	var best pmmCopy
	for _, c := range copies {
		if c.uuid != "" && (best.uuid == "" || c.rows > best.rows || (c.rows == best.rows && c.uuid < best.uuid)) {
			best = c
		}
	}
	return best.uuid
}

func readCopy(ctx context.Context, node chNode, db string) (pmmCopy, error) {
	var c pmmCopy
	err := node.db.GetContext(ctx, &c.uuid, "SELECT toString(uuid) FROM system.databases WHERE name = ?", db)
	if errors.Is(err, sql.ErrNoRows) {
		return c, nil
	}
	if err != nil {
		return c, err
	}

	err = node.db.GetContext(ctx, &c.rows, "SELECT count() FROM "+db+".metrics")
	exception, ok := errors.AsType[*clickhouse.Exception](err)
	if ok && exception.Code == unknownTableErrorCode {
		return c, nil
	}
	return c, err
}

// fixPmm brings every node onto one pmm database: it creates pmm where it is missing and, with canDrop
// (the HA leader only), replaces every other copy with the one with the most rows. See PMM-15785.
func fixPmm(ctx context.Context, nodes []chNode, db string, canDrop bool) error {
	l := logrus.WithField("component", "pmm-db")
	copies := make([]pmmCopy, len(nodes))
	for i, node := range nodes {
		c, err := readCopy(ctx, node, db)
		if err != nil {
			return fmt.Errorf("%s does not answer, changing nothing: %w", node.host, err)
		}
		copies[i] = c
	}

	kept := keptUUID(copies)
	if kept == "" {
		// No node has pmm: the first node's UUID wins, so qan-api2s starting together all use the same one.
		_, err := nodes[0].db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s UUID '%s' %s", db, uuid.NewString(), replicatedEngine))
		if err != nil {
			return err
		}
		copies[0], err = readCopy(ctx, nodes[0], db)
		if err != nil {
			return err
		}
		kept = copies[0].uuid
	}

	keeper := nodes[slices.IndexFunc(copies, func(c pmmCopy) bool { return c.uuid == kept })]
	for i, c := range copies {
		if c.uuid == kept || (c.uuid != "" && !canDrop) {
			continue
		}
		ll := l.WithFields(logrus.Fields{"node": nodes[i].host, "uuid": kept, "old_uuid": c.uuid, "dropped_rows": c.rows})
		err := replaceCopy(ctx, nodes[i], keeper, db, kept, c.uuid != "")
		if err != nil {
			ll.WithError(err).Error("Failed to recreate the pmm database.")
			continue
		}
		if c.uuid == "" {
			ll.Warn("Created the missing pmm database with the UUID of the copy with the most rows.")
			continue
		}
		ll.Warn("Recreated the pmm database with the UUID of the copy with the most rows.")
	}
	return nil
}

// replaceCopy drops target's pmm (if drop) and creates pmm with id, clearing stale Keeper entries through keeper.
func replaceCopy(ctx context.Context, target, keeper chNode, db, id string, drop bool) error {
	if drop {
		// Another pass (a new leader) may have changed the kept copy since this one read it.
		k, err := readCopy(ctx, keeper, db)
		if err != nil {
			return err
		}
		if k.uuid != id {
			return fmt.Errorf("the kept copy on %s changed to '%s', not dropping", keeper.host, k.uuid)
		}
		_, err = target.db.ExecContext(ctx, "DROP DATABASE "+db+" SYNC")
		if err != nil {
			return err
		}
	}

	create := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s UUID '%s' %s", db, id, replicatedEngine)
	_, err := target.db.ExecContext(ctx, create)
	exception, ok := errors.AsType[*clickhouse.Exception](err)
	if ok && exception.Code == replicaAlreadyExistsErrorCode {
		// After a volume loss Keeper still lists the node: drop its stale entries and create again.
		var shard, replica string
		err = target.db.QueryRowxContext(ctx, "SELECT getMacro('shard'), getMacro('replica')").Scan(&shard, &replica)
		if err != nil {
			return err
		}
		_, _ = keeper.db.ExecContext(ctx, fmt.Sprintf("SYSTEM DROP DATABASE REPLICA '%s' FROM SHARD '%s' FROM DATABASE %s", replica, shard, db))
		_, _ = keeper.db.ExecContext(ctx, fmt.Sprintf("SYSTEM DROP REPLICA '%s' FROM DATABASE %s", replica, db))
		_, err = target.db.ExecContext(ctx, create)
	}
	if err != nil {
		return err
	}

	// Wait until the new replica has created the tables, so migrations do not race it.
	_, err = target.db.ExecContext(ctx, "SYSTEM SYNC DATABASE REPLICA "+db)
	return err
}

// isLeader asks pmm-managed's HA API at url whether nodeID is the HA leader.
func isLeader(ctx context.Context, url, nodeID string) (bool, error) {
	if nodeID == "" {
		return false, errors.New("PMM_HA_NODE_ID is not set")
	}
	ctx, cancel := context.WithTimeout(ctx, leaderCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%s: %s", url, resp.Status)
	}

	var body struct {
		Nodes []struct {
			NodeName string `json:"node_name"`
			Role     string `json:"role"`
		} `json:"nodes"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return false, err
	}

	for _, n := range body.Nodes {
		if n.NodeName == nodeID && n.Role == "NODE_ROLE_LEADER" {
			return true, nil
		}
	}
	return false, nil
}

// runPmmRepairLoop lets the HA leader run fixPmm at start and then every pmmRepairInterval.
func runPmmRepairLoop(ctx context.Context, nodes []chNode, db string) {
	l := logrus.WithField("component", "pmm-db")
	for {
		leader, err := isLeader(ctx, haNodesURL, os.Getenv("PMM_HA_NODE_ID"))
		if err != nil {
			l.WithError(err).Warn("Cannot tell whether this node is the HA leader; skipped the pmm database check.")
		}
		if leader {
			err = fixPmm(ctx, nodes, db, true)
			if err != nil {
				l.WithError(err).Warn("Skipped the pmm database check.")
			}
		}

		t := time.NewTimer(pmmRepairInterval)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}
