// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Cluster Statement (support pbrown's cluster index implementation)
 * ----------------------
 */
type ClusterStmt struct {
	Relation  *RangeVar `json:"relation"`  /* relation being indexed, or NULL if all */
	Indexname *string   `json:"indexname"` /* original index defined */
	Verbose   bool      `json:"verbose"`   /* print progress info */
}

func (node ClusterStmt) MarshalJSON() ([]byte, error) {
	type ClusterStmtMarshalAlias ClusterStmt
	return json.Marshal(map[string]interface{}{
		"ClusterStmt": (*ClusterStmtMarshalAlias)(&node),
	})
}

func (node *ClusterStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["indexname"] != nil {
		err = json.Unmarshal(fields["indexname"], &node.Indexname)
		if err != nil {
			return
		}
	}

	if fields["verbose"] != nil {
		err = json.Unmarshal(fields["verbose"], &node.Verbose)
		if err != nil {
			return
		}
	}

	return
}
