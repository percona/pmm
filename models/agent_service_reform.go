package models

// generated with gopkg.in/reform.v1

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type agentServiceViewType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *agentServiceViewType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("agent_services").
func (v *agentServiceViewType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *agentServiceViewType) Columns() []string {
	return []string{"agent_id", "service_id"}
}

// NewStruct makes a new struct for that view or table.
func (v *agentServiceViewType) NewStruct() reform.Struct {
	return new(AgentService)
}

// AgentServiceView represents agent_services view or table in SQL database.
var AgentServiceView = &agentServiceViewType{
	s: parse.StructInfo{Type: "AgentService", SQLSchema: "", SQLName: "agent_services", Fields: []parse.FieldInfo{{Name: "AgentID", PKType: "", Column: "agent_id"}, {Name: "ServiceID", PKType: "", Column: "service_id"}}, PKFieldIndex: -1},
	z: new(AgentService).Values(),
}

// String returns a string representation of this struct or record.
func (s AgentService) String() string {
	res := make([]string, 2)
	res[0] = "AgentID: " + reform.Inspect(s.AgentID, true)
	res[1] = "ServiceID: " + reform.Inspect(s.ServiceID, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *AgentService) Values() []interface{} {
	return []interface{}{
		s.AgentID,
		s.ServiceID,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *AgentService) Pointers() []interface{} {
	return []interface{}{
		&s.AgentID,
		&s.ServiceID,
	}
}

// View returns View object for that struct.
func (s *AgentService) View() reform.View {
	return AgentServiceView
}

// check interfaces
var (
	_ reform.View   = AgentServiceView
	_ reform.Struct = new(AgentService)
	_ fmt.Stringer  = new(AgentService)
)

func init() {
	parse.AssertUpToDate(&AgentServiceView.s, new(AgentService))
}
