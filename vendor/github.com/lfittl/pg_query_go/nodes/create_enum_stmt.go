// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Type Statement, enum types
 * ----------------------
 */
type CreateEnumStmt struct {
	TypeName List `json:"typeName"` /* qualified name (list of Value strings) */
	Vals     List `json:"vals"`     /* enum values (list of Value strings) */
}

func (node CreateEnumStmt) MarshalJSON() ([]byte, error) {
	type CreateEnumStmtMarshalAlias CreateEnumStmt
	return json.Marshal(map[string]interface{}{
		"CreateEnumStmt": (*CreateEnumStmtMarshalAlias)(&node),
	})
}

func (node *CreateEnumStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["typeName"] != nil {
		node.TypeName.Items, err = UnmarshalNodeArrayJSON(fields["typeName"])
		if err != nil {
			return
		}
	}

	if fields["vals"] != nil {
		node.Vals.Items, err = UnmarshalNodeArrayJSON(fields["vals"])
		if err != nil {
			return
		}
	}

	return
}
