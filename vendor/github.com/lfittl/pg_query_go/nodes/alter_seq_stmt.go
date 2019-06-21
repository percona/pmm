// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		{Create|Alter} SEQUENCE Statement
 * ----------------------
 */
type AlterSeqStmt struct {
	Sequence    *RangeVar `json:"sequence"` /* the sequence to alter */
	Options     List      `json:"options"`
	ForIdentity bool      `json:"for_identity"`
	MissingOk   bool      `json:"missing_ok"` /* skip error if a role is missing? */
}

func (node AlterSeqStmt) MarshalJSON() ([]byte, error) {
	type AlterSeqStmtMarshalAlias AlterSeqStmt
	return json.Marshal(map[string]interface{}{
		"AlterSeqStmt": (*AlterSeqStmtMarshalAlias)(&node),
	})
}

func (node *AlterSeqStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["sequence"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["sequence"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Sequence = &val
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	if fields["for_identity"] != nil {
		err = json.Unmarshal(fields["for_identity"], &node.ForIdentity)
		if err != nil {
			return
		}
	}

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	return
}
