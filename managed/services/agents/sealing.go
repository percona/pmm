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

package agents

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/vault"
)

// SealingKeyResponse is the response of SealingKeyHandler.
type SealingKeyResponse struct {
	PMMAgentID string `json:"pmm_agent_id"`
	// PublicKey is an uncompressed P-256 point in standard base64.
	PublicKey string `json:"public_key"`
	Algorithm string `json:"algorithm"`
}

// SealingKeyHandler answers with the key that the pmm-agent of the Agent with the ID in the path seals secrets to, so
// that a client seals them for that pmm-agent alone, and PMM Server passes them on unread, see vault.Seal. The ID
// may be the one of the pmm-agent itself.
func (r *Registry) SealingKeyHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		id := req.PathValue("agent_id")
		agent, err := models.FindAgentByID(r.db.Querier, id)
		if status.Code(err) == codes.NotFound {
			writeSealingError(rw, http.StatusNotFound, "Agent with ID "+id+" not found.")
			return
		}
		if err != nil {
			writeSealingError(rw, http.StatusInternalServerError, "Cannot find Agent with ID "+id+".")
			logrus.WithField("component", "agents").Errorf("Cannot find Agent with ID %s: %s.", id, err)
			return
		}
		pmmAgentID := id
		if agent.AgentType != models.PMMAgentType {
			pmmAgentID = pointer.GetString(agent.PMMAgentID)
		}

		info, err := r.get(pmmAgentID)
		if err != nil {
			writeSealingError(rw, http.StatusConflict, "pmm-agent with ID "+pmmAgentID+" is not connected.")
			return
		}
		if info.sealingKey == "" {
			writeSealingError(rw, http.StatusConflict, "pmm-agent with ID "+pmmAgentID+" does not seal secrets, upgrade it.")
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(rw).Encode(SealingKeyResponse{
			PMMAgentID: pmmAgentID,
			PublicKey:  info.sealingKey,
			Algorithm:  vault.SealingAlgorithm,
		})
		if err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
			logrus.WithField("component", "agents").Debugf("Cannot write the sealing key: %s.", err)
		}
	})
}

func writeSealingError(rw http.ResponseWriter, status int, message string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	err := json.NewEncoder(rw).Encode(map[string]string{"message": message})
	if err != nil {
		logrus.WithField("component", "agents").Debugf("Cannot write the error: %s.", err)
	}
}
