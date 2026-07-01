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
	"encoding/json"
	"strings"

	"github.com/percona/pmm/managed/models"
)

const configKeyUserRequest = "user_request"

func investigationConfigMap(inv *models.Investigation) map[string]any {
	cfg := map[string]any{}
	if len(inv.Config) > 0 {
		_ = json.Unmarshal(inv.Config, &cfg)
	}
	return cfg
}

func userRequestFromInvestigation(inv *models.Investigation) string {
	if v, _ := investigationConfigMap(inv)[configKeyUserRequest].(string); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return ""
}

func setUserRequestInConfig(inv *models.Investigation, userRequest string) {
	userRequest = strings.TrimSpace(userRequest)
	if userRequest == "" {
		return
	}
	cfg := investigationConfigMap(inv)
	if existing, _ := cfg[configKeyUserRequest].(string); strings.TrimSpace(existing) != "" {
		return
	}
	cfg[configKeyUserRequest] = userRequest
	b, err := json.Marshal(cfg)
	if err == nil {
		inv.Config = b
	}
}

// preserveUserRequest copies inv.Summary into config before report generation overwrites it.
func preserveUserRequest(inv *models.Investigation) {
	setUserRequestInConfig(inv, inv.Summary)
}
