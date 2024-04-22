// Copyright (C) 2024 Percona LLC
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

package services

// This file contains grafana alerting API DTOs.

// Rule represents grafana alerting rule.
type Rule struct {
	GrafanaAlert GrafanaAlert      `json:"grafana_alert"`
	For          string            `json:"for"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
}

// RelativeTimeRange defines grafana API time range.
type RelativeTimeRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// Model represents grafana query model.
type Model struct {
	RefID   string `json:"refId"`
	Expr    string `json:"expr"`
	Instant bool   `json:"instant"`
}

// Data represents grafana API alert rule data.
type Data struct {
	RefID             string            `json:"refId"`
	DatasourceUID     string            `json:"datasourceUid"`
	QueryType         string            `json:"queryType"`
	RelativeTimeRange RelativeTimeRange `json:"relativeTimeRange,omitempty"`
	Model             Model             `json:"model,omitempty"`
}

// GrafanaAlert represent grafana alerting rule.
type GrafanaAlert struct {
	Title        string `json:"title"`
	Condition    string `json:"condition"`
	NoDataState  string `json:"no_data_state"`
	ExecErrState string `json:"exec_err_state"`
	Data         []Data `json:"data"`
}
