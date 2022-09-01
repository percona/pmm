// Copyright (C) 2017 Percona LLC
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

type Rule struct {
	GrafanaAlert GrafanaAlert      `json:"grafana_alert"`
	For          string            `json:"for"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
}

type RelativeTimeRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}
type Model struct {
	RefID   string `json:"refId"`
	Expr    string `json:"expr"`
	Instant bool   `json:"instant"`
}
type Datasource struct {
	UID  string `json:"uid"`
	Type string `json:"type"`
}
type Evaluator struct {
	Params []int  `json:"params"`
	Type   string `json:"type"`
}
type Operator struct {
	Type string `json:"type"`
}
type Query struct {
	Params []string `json:"params"`
}
type Reducer struct {
	Params []interface{} `json:"params"`
	Type   string        `json:"type"`
}
type Condition struct {
	Type      string    `json:"type"`
	Evaluator Evaluator `json:"evaluator"`
	Operator  Operator  `json:"operator"`
	Query     Query     `json:"query"`
	Reducer   Reducer   `json:"reducer"`
}

type Data struct {
	RefID             string            `json:"refId"`
	DatasourceUID     string            `json:"datasourceUid"`
	QueryType         string            `json:"queryType"`
	RelativeTimeRange RelativeTimeRange `json:"relativeTimeRange,omitempty"`
	Model             Model             `json:"model,omitempty"`
}
type GrafanaAlert struct {
	Title        string `json:"title"`
	Condition    string `json:"condition"`
	NoDataState  string `json:"no_data_state"`
	ExecErrState string `json:"exec_err_state"`
	Data         []Data `json:"data"`
}
