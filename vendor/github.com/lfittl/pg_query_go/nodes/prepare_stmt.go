// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		PREPARE Statement
 * ----------------------
 */
type PrepareStmt struct {
	Name     *string `json:"name"`     /* Name of plan, arbitrary */
	Argtypes List    `json:"argtypes"` /* Types of parameters (List of TypeName) */
	Query    Node    `json:"query"`    /* The query itself (as a raw parsetree) */
}

func (node PrepareStmt) MarshalJSON() ([]byte, error) {
	type PrepareStmtMarshalAlias PrepareStmt
	return json.Marshal(map[string]interface{}{
		"PrepareStmt": (*PrepareStmtMarshalAlias)(&node),
	})
}

func (node *PrepareStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["argtypes"] != nil {
		node.Argtypes.Items, err = UnmarshalNodeArrayJSON(fields["argtypes"])
		if err != nil {
			return
		}
	}

	if fields["query"] != nil {
		node.Query, err = UnmarshalNodeJSON(fields["query"])
		if err != nil {
			return
		}
	}

	return
}
