// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 * Alter Collation
 * ----------------------
 */
type AlterCollationStmt struct {
	Collname List `json:"collname"`
}

func (node AlterCollationStmt) MarshalJSON() ([]byte, error) {
	type AlterCollationStmtMarshalAlias AlterCollationStmt
	return json.Marshal(map[string]interface{}{
		"AlterCollationStmt": (*AlterCollationStmtMarshalAlias)(&node),
	})
}

func (node *AlterCollationStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["collname"] != nil {
		node.Collname.Items, err = UnmarshalNodeArrayJSON(fields["collname"])
		if err != nil {
			return
		}
	}

	return
}
