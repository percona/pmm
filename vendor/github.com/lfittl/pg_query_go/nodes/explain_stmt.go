// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Explain Statement
 *
 * The "query" field is initially a raw parse tree, and is converted to a
 * Query node during parse analysis.  Note that rewriting and planning
 * of the query are always postponed until execution.
 * ----------------------
 */
type ExplainStmt struct {
	Query   Node `json:"query"`   /* the query (see comments above) */
	Options List `json:"options"` /* list of DefElem nodes */
}

func (node ExplainStmt) MarshalJSON() ([]byte, error) {
	type ExplainStmtMarshalAlias ExplainStmt
	return json.Marshal(map[string]interface{}{
		"ExplainStmt": (*ExplainStmtMarshalAlias)(&node),
	})
}

func (node *ExplainStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["query"] != nil {
		node.Query, err = UnmarshalNodeJSON(fields["query"])
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
