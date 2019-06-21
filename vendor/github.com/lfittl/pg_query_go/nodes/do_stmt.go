// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		DO Statement
 *
 * DoStmt is the raw parser output, InlineCodeBlock is the execution-time API
 * ----------------------
 */
type DoStmt struct {
	Args List `json:"args"` /* List of DefElem nodes */
}

func (node DoStmt) MarshalJSON() ([]byte, error) {
	type DoStmtMarshalAlias DoStmt
	return json.Marshal(map[string]interface{}{
		"DoStmt": (*DoStmtMarshalAlias)(&node),
	})
}

func (node *DoStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	return
}
