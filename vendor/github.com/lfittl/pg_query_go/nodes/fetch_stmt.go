// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Fetch Statement (also Move)
 * ----------------------
 */
type FetchStmt struct {
	Direction  FetchDirection `json:"direction"`  /* see above */
	HowMany    int64          `json:"howMany"`    /* number of rows, or position argument */
	Portalname *string        `json:"portalname"` /* name of portal (cursor) */
	Ismove     bool           `json:"ismove"`     /* TRUE if MOVE */
}

func (node FetchStmt) MarshalJSON() ([]byte, error) {
	type FetchStmtMarshalAlias FetchStmt
	return json.Marshal(map[string]interface{}{
		"FetchStmt": (*FetchStmtMarshalAlias)(&node),
	})
}

func (node *FetchStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["direction"] != nil {
		err = json.Unmarshal(fields["direction"], &node.Direction)
		if err != nil {
			return
		}
	}

	if fields["howMany"] != nil {
		err = json.Unmarshal(fields["howMany"], &node.HowMany)
		if err != nil {
			return
		}
	}

	if fields["portalname"] != nil {
		err = json.Unmarshal(fields["portalname"], &node.Portalname)
		if err != nil {
			return
		}
	}

	if fields["ismove"] != nil {
		err = json.Unmarshal(fields["ismove"], &node.Ismove)
		if err != nil {
			return
		}
	}

	return
}
