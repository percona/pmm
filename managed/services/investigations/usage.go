// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package investigations

import (
	"net/http"
	"strings"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

// GetInvestigationUsage handles GET /v1/investigations/:id/usage.
func (h *Handlers) GetInvestigationUsage(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "investigation id is required")
		return
	}
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationByID: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	events, err := adre.QueryInvestigationUsageEvents(h.db, id)
	if err != nil {
		h.l.Errorf("QueryInvestigationUsageEvents: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load investigation usage")
		return
	}
	adre.WriteInvestigationUsageResponse(w, id, events)
}
