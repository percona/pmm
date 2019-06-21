// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TableLikeClause - CREATE TABLE ( ... LIKE ... ) clause
 */
type TableLikeClause struct {
	Relation *RangeVar `json:"relation"`
	Options  uint32    `json:"options"` /* OR of TableLikeOption flags */
}

func (node TableLikeClause) MarshalJSON() ([]byte, error) {
	type TableLikeClauseMarshalAlias TableLikeClause
	return json.Marshal(map[string]interface{}{
		"TableLikeClause": (*TableLikeClauseMarshalAlias)(&node),
	})
}

func (node *TableLikeClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
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

	if fields["options"] != nil {
		err = json.Unmarshal(fields["options"], &node.Options)
		if err != nil {
			return
		}
	}

	return
}
