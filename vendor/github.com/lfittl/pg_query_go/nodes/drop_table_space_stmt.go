// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Drop Table Space Statements
 * ----------------------
 */
type DropTableSpaceStmt struct {
	Tablespacename *string `json:"tablespacename"`
	MissingOk      bool    `json:"missing_ok"` /* skip error if missing? */
}

func (node DropTableSpaceStmt) MarshalJSON() ([]byte, error) {
	type DropTableSpaceStmtMarshalAlias DropTableSpaceStmt
	return json.Marshal(map[string]interface{}{
		"DropTableSpaceStmt": (*DropTableSpaceStmtMarshalAlias)(&node),
	})
}

func (node *DropTableSpaceStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["tablespacename"] != nil {
		err = json.Unmarshal(fields["tablespacename"], &node.Tablespacename)
		if err != nil {
			return
		}
	}

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	return
}
