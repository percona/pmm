// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		{Create|Alter} SEQUENCE Statement
 * ----------------------
 */
type CreateSeqStmt struct {
	Sequence    *RangeVar `json:"sequence"` /* the sequence to create */
	Options     List      `json:"options"`
	OwnerId     Oid       `json:"ownerId"` /* ID of owner, or InvalidOid for default */
	ForIdentity bool      `json:"for_identity"`
	IfNotExists bool      `json:"if_not_exists"` /* just do nothing if it already exists? */
}

func (node CreateSeqStmt) MarshalJSON() ([]byte, error) {
	type CreateSeqStmtMarshalAlias CreateSeqStmt
	return json.Marshal(map[string]interface{}{
		"CreateSeqStmt": (*CreateSeqStmtMarshalAlias)(&node),
	})
}

func (node *CreateSeqStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["ownerId"] != nil {
		err = json.Unmarshal(fields["ownerId"], &node.OwnerId)
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

	if fields["if_not_exists"] != nil {
		err = json.Unmarshal(fields["if_not_exists"], &node.IfNotExists)
		if err != nil {
			return
		}
	}

	return
}
