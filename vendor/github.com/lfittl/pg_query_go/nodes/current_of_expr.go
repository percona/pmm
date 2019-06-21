// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Node representing [WHERE] CURRENT OF cursor_name
 *
 * CURRENT OF is a bit like a Var, in that it carries the rangetable index
 * of the target relation being constrained; this aids placing the expression
 * correctly during planning.  We can assume however that its "levelsup" is
 * always zero, due to the syntactic constraints on where it can appear.
 *
 * The referenced cursor can be represented either as a hardwired string
 * or as a reference to a run-time parameter of type REFCURSOR.  The latter
 * case is for the convenience of plpgsql.
 */
type CurrentOfExpr struct {
	Xpr         Node    `json:"xpr"`
	Cvarno      Index   `json:"cvarno"`       /* RT index of target relation */
	CursorName  *string `json:"cursor_name"`  /* name of referenced cursor, or NULL */
	CursorParam int     `json:"cursor_param"` /* refcursor parameter number, or 0 */
}

func (node CurrentOfExpr) MarshalJSON() ([]byte, error) {
	type CurrentOfExprMarshalAlias CurrentOfExpr
	return json.Marshal(map[string]interface{}{
		"CurrentOfExpr": (*CurrentOfExprMarshalAlias)(&node),
	})
}

func (node *CurrentOfExpr) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xpr"] != nil {
		node.Xpr, err = UnmarshalNodeJSON(fields["xpr"])
		if err != nil {
			return
		}
	}

	if fields["cvarno"] != nil {
		err = json.Unmarshal(fields["cvarno"], &node.Cvarno)
		if err != nil {
			return
		}
	}

	if fields["cursor_name"] != nil {
		err = json.Unmarshal(fields["cursor_name"], &node.CursorName)
		if err != nil {
			return
		}
	}

	if fields["cursor_param"] != nil {
		err = json.Unmarshal(fields["cursor_param"], &node.CursorParam)
		if err != nil {
			return
		}
	}

	return
}
