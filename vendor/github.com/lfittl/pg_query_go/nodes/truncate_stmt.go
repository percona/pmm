// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *				Truncate Table Statement
 * ----------------------
 */
type TruncateStmt struct {
	Relations   List         `json:"relations"`    /* relations (RangeVars) to be truncated */
	RestartSeqs bool         `json:"restart_seqs"` /* restart owned sequences? */
	Behavior    DropBehavior `json:"behavior"`     /* RESTRICT or CASCADE behavior */
}

func (node TruncateStmt) MarshalJSON() ([]byte, error) {
	type TruncateStmtMarshalAlias TruncateStmt
	return json.Marshal(map[string]interface{}{
		"TruncateStmt": (*TruncateStmtMarshalAlias)(&node),
	})
}

func (node *TruncateStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["restart_seqs"] != nil {
		err = json.Unmarshal(fields["restart_seqs"], &node.RestartSeqs)
		if err != nil {
			return
		}
	}

	if fields["behavior"] != nil {
		err = json.Unmarshal(fields["behavior"], &node.Behavior)
		if err != nil {
			return
		}
	}

	return
}
