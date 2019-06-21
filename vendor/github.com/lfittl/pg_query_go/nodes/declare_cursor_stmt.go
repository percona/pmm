// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* these planner-control flags do not correspond to any SQL grammar: */
type DeclareCursorStmt struct {
	Portalname *string `json:"portalname"` /* name of the portal (cursor) */
	Options    int     `json:"options"`    /* bitmask of options (see above) */
	Query      Node    `json:"query"`      /* the query (see comments above) */
}

func (node DeclareCursorStmt) MarshalJSON() ([]byte, error) {
	type DeclareCursorStmtMarshalAlias DeclareCursorStmt
	return json.Marshal(map[string]interface{}{
		"DeclareCursorStmt": (*DeclareCursorStmtMarshalAlias)(&node),
	})
}

func (node *DeclareCursorStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["portalname"] != nil {
		err = json.Unmarshal(fields["portalname"], &node.Portalname)
		if err != nil {
			return
		}
	}

	if fields["options"] != nil {
		err = json.Unmarshal(fields["options"], &node.Options)
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
