// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Createdb Statement
 * ----------------------
 */
type CreatedbStmt struct {
	Dbname  *string `json:"dbname"`  /* name of database to create */
	Options List    `json:"options"` /* List of DefElem nodes */
}

func (node CreatedbStmt) MarshalJSON() ([]byte, error) {
	type CreatedbStmtMarshalAlias CreatedbStmt
	return json.Marshal(map[string]interface{}{
		"CreatedbStmt": (*CreatedbStmtMarshalAlias)(&node),
	})
}

func (node *CreatedbStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["dbname"] != nil {
		err = json.Unmarshal(fields["dbname"], &node.Dbname)
		if err != nil {
			return
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	return
}
