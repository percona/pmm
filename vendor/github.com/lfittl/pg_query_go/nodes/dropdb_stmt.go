// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Dropdb Statement
 * ----------------------
 */
type DropdbStmt struct {
	Dbname    *string `json:"dbname"`     /* database to drop */
	MissingOk bool    `json:"missing_ok"` /* skip error if db is missing? */
}

func (node DropdbStmt) MarshalJSON() ([]byte, error) {
	type DropdbStmtMarshalAlias DropdbStmt
	return json.Marshal(map[string]interface{}{
		"DropdbStmt": (*DropdbStmtMarshalAlias)(&node),
	})
}

func (node *DropdbStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	return
}
