package models

// generated with gopkg.in/reform.v1

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type agentTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *agentTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("agents").
func (v *agentTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *agentTableType) Columns() []string {
	return []string{"id", "type", "runs_on_node_id"}
}

// NewStruct makes a new struct for that view or table.
func (v *agentTableType) NewStruct() reform.Struct {
	return new(Agent)
}

// NewRecord makes a new record for that table.
func (v *agentTableType) NewRecord() reform.Record {
	return new(Agent)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *agentTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// AgentTable represents agents view or table in SQL database.
var AgentTable = &agentTableType{
	s: parse.StructInfo{Type: "Agent", SQLSchema: "", SQLName: "agents", Fields: []parse.FieldInfo{{Name: "ID", PKType: "int32", Column: "id"}, {Name: "Type", PKType: "", Column: "type"}, {Name: "RunsOnNodeID", PKType: "", Column: "runs_on_node_id"}}, PKFieldIndex: 0},
	z: new(Agent).Values(),
}

// String returns a string representation of this struct or record.
func (s Agent) String() string {
	res := make([]string, 3)
	res[0] = "ID: " + reform.Inspect(s.ID, true)
	res[1] = "Type: " + reform.Inspect(s.Type, true)
	res[2] = "RunsOnNodeID: " + reform.Inspect(s.RunsOnNodeID, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *Agent) Values() []interface{} {
	return []interface{}{
		s.ID,
		s.Type,
		s.RunsOnNodeID,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *Agent) Pointers() []interface{} {
	return []interface{}{
		&s.ID,
		&s.Type,
		&s.RunsOnNodeID,
	}
}

// View returns View object for that struct.
func (s *Agent) View() reform.View {
	return AgentTable
}

// Table returns Table object for that record.
func (s *Agent) Table() reform.Table {
	return AgentTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *Agent) PKValue() interface{} {
	return s.ID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *Agent) PKPointer() interface{} {
	return &s.ID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *Agent) HasPK() bool {
	return s.ID != AgentTable.z[AgentTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *Agent) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.ID = int32(i64)
	} else {
		s.ID = pk.(int32)
	}
}

// check interfaces
var (
	_ reform.View   = AgentTable
	_ reform.Struct = new(Agent)
	_ reform.Table  = AgentTable
	_ reform.Record = new(Agent)
	_ fmt.Stringer  = new(Agent)
)

type mySQLdExporterTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *mySQLdExporterTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("agents").
func (v *mySQLdExporterTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *mySQLdExporterTableType) Columns() []string {
	return []string{"id", "type", "runs_on_node_id", "service_username", "service_password"}
}

// NewStruct makes a new struct for that view or table.
func (v *mySQLdExporterTableType) NewStruct() reform.Struct {
	return new(MySQLdExporter)
}

// NewRecord makes a new record for that table.
func (v *mySQLdExporterTableType) NewRecord() reform.Record {
	return new(MySQLdExporter)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *mySQLdExporterTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// MySQLdExporterTable represents agents view or table in SQL database.
var MySQLdExporterTable = &mySQLdExporterTableType{
	s: parse.StructInfo{Type: "MySQLdExporter", SQLSchema: "", SQLName: "agents", Fields: []parse.FieldInfo{{Name: "ID", PKType: "int32", Column: "id"}, {Name: "Type", PKType: "", Column: "type"}, {Name: "RunsOnNodeID", PKType: "", Column: "runs_on_node_id"}, {Name: "ServiceUsername", PKType: "", Column: "service_username"}, {Name: "ServicePassword", PKType: "", Column: "service_password"}}, PKFieldIndex: 0},
	z: new(MySQLdExporter).Values(),
}

// String returns a string representation of this struct or record.
func (s MySQLdExporter) String() string {
	res := make([]string, 5)
	res[0] = "ID: " + reform.Inspect(s.ID, true)
	res[1] = "Type: " + reform.Inspect(s.Type, true)
	res[2] = "RunsOnNodeID: " + reform.Inspect(s.RunsOnNodeID, true)
	res[3] = "ServiceUsername: " + reform.Inspect(s.ServiceUsername, true)
	res[4] = "ServicePassword: " + reform.Inspect(s.ServicePassword, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *MySQLdExporter) Values() []interface{} {
	return []interface{}{
		s.ID,
		s.Type,
		s.RunsOnNodeID,
		s.ServiceUsername,
		s.ServicePassword,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *MySQLdExporter) Pointers() []interface{} {
	return []interface{}{
		&s.ID,
		&s.Type,
		&s.RunsOnNodeID,
		&s.ServiceUsername,
		&s.ServicePassword,
	}
}

// View returns View object for that struct.
func (s *MySQLdExporter) View() reform.View {
	return MySQLdExporterTable
}

// Table returns Table object for that record.
func (s *MySQLdExporter) Table() reform.Table {
	return MySQLdExporterTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *MySQLdExporter) PKValue() interface{} {
	return s.ID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *MySQLdExporter) PKPointer() interface{} {
	return &s.ID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *MySQLdExporter) HasPK() bool {
	return s.ID != MySQLdExporterTable.z[MySQLdExporterTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *MySQLdExporter) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.ID = int32(i64)
	} else {
		s.ID = pk.(int32)
	}
}

// check interfaces
var (
	_ reform.View   = MySQLdExporterTable
	_ reform.Struct = new(MySQLdExporter)
	_ reform.Table  = MySQLdExporterTable
	_ reform.Record = new(MySQLdExporter)
	_ fmt.Stringer  = new(MySQLdExporter)
)

func init() {
	parse.AssertUpToDate(&AgentTable.s, new(Agent))
	parse.AssertUpToDate(&MySQLdExporterTable.s, new(MySQLdExporter))
}
