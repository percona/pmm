package models

// generated with gopkg.in/reform.v1

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type serviceTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *serviceTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("services").
func (v *serviceTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *serviceTableType) Columns() []string {
	return []string{"id", "type", "node_id"}
}

// NewStruct makes a new struct for that view or table.
func (v *serviceTableType) NewStruct() reform.Struct {
	return new(Service)
}

// NewRecord makes a new record for that table.
func (v *serviceTableType) NewRecord() reform.Record {
	return new(Service)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *serviceTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// ServiceTable represents services view or table in SQL database.
var ServiceTable = &serviceTableType{
	s: parse.StructInfo{Type: "Service", SQLSchema: "", SQLName: "services", Fields: []parse.FieldInfo{{Name: "ID", PKType: "int64", Column: "id"}, {Name: "Type", PKType: "", Column: "type"}, {Name: "NodeID", PKType: "", Column: "node_id"}}, PKFieldIndex: 0},
	z: new(Service).Values(),
}

// String returns a string representation of this struct or record.
func (s Service) String() string {
	res := make([]string, 3)
	res[0] = "ID: " + reform.Inspect(s.ID, true)
	res[1] = "Type: " + reform.Inspect(s.Type, true)
	res[2] = "NodeID: " + reform.Inspect(s.NodeID, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *Service) Values() []interface{} {
	return []interface{}{
		s.ID,
		s.Type,
		s.NodeID,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *Service) Pointers() []interface{} {
	return []interface{}{
		&s.ID,
		&s.Type,
		&s.NodeID,
	}
}

// View returns View object for that struct.
func (s *Service) View() reform.View {
	return ServiceTable
}

// Table returns Table object for that record.
func (s *Service) Table() reform.Table {
	return ServiceTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *Service) PKValue() interface{} {
	return s.ID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *Service) PKPointer() interface{} {
	return &s.ID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *Service) HasPK() bool {
	return s.ID != ServiceTable.z[ServiceTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *Service) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.ID = int64(i64)
	} else {
		s.ID = pk.(int64)
	}
}

// check interfaces
var (
	_ reform.View   = ServiceTable
	_ reform.Struct = new(Service)
	_ reform.Table  = ServiceTable
	_ reform.Record = new(Service)
	_ fmt.Stringer  = new(Service)
)

type rDSServiceTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *rDSServiceTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("services").
func (v *rDSServiceTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *rDSServiceTableType) Columns() []string {
	return []string{"id", "type", "node_id", "address", "port", "engine", "engine_version"}
}

// NewStruct makes a new struct for that view or table.
func (v *rDSServiceTableType) NewStruct() reform.Struct {
	return new(RDSService)
}

// NewRecord makes a new record for that table.
func (v *rDSServiceTableType) NewRecord() reform.Record {
	return new(RDSService)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *rDSServiceTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// RDSServiceTable represents services view or table in SQL database.
var RDSServiceTable = &rDSServiceTableType{
	s: parse.StructInfo{Type: "RDSService", SQLSchema: "", SQLName: "services", Fields: []parse.FieldInfo{{Name: "ID", PKType: "int64", Column: "id"}, {Name: "Type", PKType: "", Column: "type"}, {Name: "NodeID", PKType: "", Column: "node_id"}, {Name: "Address", PKType: "", Column: "address"}, {Name: "Port", PKType: "", Column: "port"}, {Name: "Engine", PKType: "", Column: "engine"}, {Name: "EngineVersion", PKType: "", Column: "engine_version"}}, PKFieldIndex: 0},
	z: new(RDSService).Values(),
}

// String returns a string representation of this struct or record.
func (s RDSService) String() string {
	res := make([]string, 7)
	res[0] = "ID: " + reform.Inspect(s.ID, true)
	res[1] = "Type: " + reform.Inspect(s.Type, true)
	res[2] = "NodeID: " + reform.Inspect(s.NodeID, true)
	res[3] = "Address: " + reform.Inspect(s.Address, true)
	res[4] = "Port: " + reform.Inspect(s.Port, true)
	res[5] = "Engine: " + reform.Inspect(s.Engine, true)
	res[6] = "EngineVersion: " + reform.Inspect(s.EngineVersion, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *RDSService) Values() []interface{} {
	return []interface{}{
		s.ID,
		s.Type,
		s.NodeID,
		s.Address,
		s.Port,
		s.Engine,
		s.EngineVersion,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *RDSService) Pointers() []interface{} {
	return []interface{}{
		&s.ID,
		&s.Type,
		&s.NodeID,
		&s.Address,
		&s.Port,
		&s.Engine,
		&s.EngineVersion,
	}
}

// View returns View object for that struct.
func (s *RDSService) View() reform.View {
	return RDSServiceTable
}

// Table returns Table object for that record.
func (s *RDSService) Table() reform.Table {
	return RDSServiceTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *RDSService) PKValue() interface{} {
	return s.ID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *RDSService) PKPointer() interface{} {
	return &s.ID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *RDSService) HasPK() bool {
	return s.ID != RDSServiceTable.z[RDSServiceTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *RDSService) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.ID = int64(i64)
	} else {
		s.ID = pk.(int64)
	}
}

// check interfaces
var (
	_ reform.View   = RDSServiceTable
	_ reform.Struct = new(RDSService)
	_ reform.Table  = RDSServiceTable
	_ reform.Record = new(RDSService)
	_ fmt.Stringer  = new(RDSService)
)

func init() {
	parse.AssertUpToDate(&ServiceTable.s, new(Service))
	parse.AssertUpToDate(&RDSServiceTable.s, new(RDSService))
}
