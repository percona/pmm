// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type ReindexStmt struct {
	Kind ReindexObjectType `json:"kind"` /* REINDEX_OBJECT_INDEX, REINDEX_OBJECT_TABLE,
	 * etc. */

	Relation *RangeVar `json:"relation"` /* Table or index to reindex */
	Name     *string   `json:"name"`     /* name of database to reindex */
	Options  int       `json:"options"`  /* Reindex options flags */
}

func (node ReindexStmt) MarshalJSON() ([]byte, error) {
	type ReindexStmtMarshalAlias ReindexStmt
	return json.Marshal(map[string]interface{}{
		"ReindexStmt": (*ReindexStmtMarshalAlias)(&node),
	})
}

func (node *ReindexStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["kind"] != nil {
		err = json.Unmarshal(fields["kind"], &node.Kind)
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

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
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

	return
}
