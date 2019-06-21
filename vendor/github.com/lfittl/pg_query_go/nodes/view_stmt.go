// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create View Statement
 * ----------------------
 */
type ViewStmt struct {
	View            *RangeVar       `json:"view"`            /* the view to be created */
	Aliases         List            `json:"aliases"`         /* target column names */
	Query           Node            `json:"query"`           /* the SELECT query (as a raw parse tree) */
	Replace         bool            `json:"replace"`         /* replace an existing view? */
	Options         List            `json:"options"`         /* options from WITH clause */
	WithCheckOption ViewCheckOption `json:"withCheckOption"` /* WITH CHECK OPTION */
}

func (node ViewStmt) MarshalJSON() ([]byte, error) {
	type ViewStmtMarshalAlias ViewStmt
	return json.Marshal(map[string]interface{}{
		"ViewStmt": (*ViewStmtMarshalAlias)(&node),
	})
}

func (node *ViewStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["view"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["view"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.View = &val
		}
	}

	if fields["aliases"] != nil {
		node.Aliases.Items, err = UnmarshalNodeArrayJSON(fields["aliases"])
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

	if fields["replace"] != nil {
		err = json.Unmarshal(fields["replace"], &node.Replace)
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

	if fields["withCheckOption"] != nil {
		err = json.Unmarshal(fields["withCheckOption"], &node.WithCheckOption)
		if err != nil {
			return
		}
	}

	return
}
