// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		LOCK Statement
 * ----------------------
 */
type LockStmt struct {
	Relations List `json:"relations"` /* relations to lock */
	Mode      int  `json:"mode"`      /* lock mode */
	Nowait    bool `json:"nowait"`    /* no wait mode */
}

func (node LockStmt) MarshalJSON() ([]byte, error) {
	type LockStmtMarshalAlias LockStmt
	return json.Marshal(map[string]interface{}{
		"LockStmt": (*LockStmtMarshalAlias)(&node),
	})
}

func (node *LockStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["relations"] != nil {
		node.Relations.Items, err = UnmarshalNodeArrayJSON(fields["relations"])
		if err != nil {
			return
		}
	}

	if fields["mode"] != nil {
		err = json.Unmarshal(fields["mode"], &node.Mode)
		if err != nil {
			return
		}
	}

	if fields["nowait"] != nil {
		err = json.Unmarshal(fields["nowait"], &node.Nowait)
		if err != nil {
			return
		}
	}

	return
}
