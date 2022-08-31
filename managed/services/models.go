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
	RefID        string      `json:"refId"`
	Hide         bool        `json:"hide"`
	EditorMode   string      `json:"editorMode"`
	Expr         string      `json:"expr"`
	LegendFormat string      `json:"legendFormat"`
	Range        bool        `json:"range"`
	Instant      bool        `json:"instant"`
	Exemplar     bool        `json:"exemplar"`
	Type         string      `json:"type"`
	Datasource   Datasource  `json:"datasource"`
	Conditions   []Condition `json:"conditions"`
	Expression   string      `json:"expression"`
	Reducer      string      `json:"reducer"`
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
