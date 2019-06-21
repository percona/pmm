// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		REFRESH MATERIALIZED VIEW Statement
 * ----------------------
 */
type RefreshMatViewStmt struct {
	Concurrent bool      `json:"concurrent"` /* allow concurrent access? */
	SkipData   bool      `json:"skipData"`   /* true for WITH NO DATA */
	Relation   *RangeVar `json:"relation"`   /* relation to insert into */
}

func (node RefreshMatViewStmt) MarshalJSON() ([]byte, error) {
	type RefreshMatViewStmtMarshalAlias RefreshMatViewStmt
	return json.Marshal(map[string]interface{}{
		"RefreshMatViewStmt": (*RefreshMatViewStmtMarshalAlias)(&node),
	})
}

func (node *RefreshMatViewStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["concurrent"] != nil {
		err = json.Unmarshal(fields["concurrent"], &node.Concurrent)
		if err != nil {
			return
		}
	}

	if fields["skipData"] != nil {
		err = json.Unmarshal(fields["skipData"], &node.SkipData)
		if err != nil {
			return
		}
	}

	if fields["relation"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["relation"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Relation = &val
		}
	}

	return
}
