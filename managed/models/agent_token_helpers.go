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
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

const (
	// AgentTokenPrefix marks a token as issued by pmm-managed rather than by Grafana.
	// AuthServer uses it to decide, with no round-trip, which credential store to consult.
	AgentTokenPrefix = "pmmat_"

	// Entropy behind a token, in bytes. Tokens are random secrets rather than passwords, so
	// they are hashed with SHA-256: a slow KDF would only add latency to a check that runs on
	// every agent request, and buys nothing against 256 bits of entropy.
	agentTokenBytes = 32
)

// ErrInvalidAgentToken is returned when a token is malformed or matches no issued token.
var ErrInvalidAgentToken = errors.New("invalid agent token")

// IsAgentToken reports whether a credential was issued by pmm-managed.
func IsAgentToken(token string) bool {
	return strings.HasPrefix(token, AgentTokenPrefix)
}

// HashAgentToken returns the stored form of a token.
func HashAgentToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateAgentToken issues a new token for a Node and returns it. The plaintext is returned
// only here; afterwards only its hash exists.
func CreateAgentToken(q *reform.Querier, nodeID string) (*AgentToken, string, error) {
	if nodeID == "" {
		return nil, "", status.Error(codes.InvalidArgument, "Empty Node ID.")
	}

	buf := make([]byte, agentTokenBytes)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, "", err
	}
	token := AgentTokenPrefix + base64.RawURLEncoding.EncodeToString(buf)

	row := &AgentToken{
		TokenHash: HashAgentToken(token),
		NodeID:    nodeID,
	}
	err = q.Insert(row)
	if err != nil {
		return nil, "", err
	}

	return row, token, nil
}

// FindNodeIDByAgentToken resolves a token to the Node it was issued for.
func FindNodeIDByAgentToken(q *reform.Querier, token string) (string, error) {
	if !IsAgentToken(token) {
		return "", ErrInvalidAgentToken
	}

	var row AgentToken
	err := q.FindOneTo(&row, "token_hash", HashAgentToken(token))
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return "", ErrInvalidAgentToken
		}
		return "", err
	}

	return row.NodeID, nil
}

// RemoveAgentTokensForNode revokes every token issued for a Node. Node removal cascades to
// the same rows; this is for revoking without unregistering, as rotation does.
func RemoveAgentTokensForNode(q *reform.Querier, nodeID string) error {
	_, err := q.DeleteFrom(AgentTokenTable, "WHERE node_id = "+q.Placeholder(1), nodeID)
	return err
}
