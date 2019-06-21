// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Vacuum and Analyze Statements
 *
 * Even though these are nominally two statements, it's convenient to use
 * just one node type for both.  Note that at least one of VACOPT_VACUUM
 * and VACOPT_ANALYZE must be set in options.
 * ----------------------
 */
type VacuumStmt struct {
	Options  int       `json:"options"`  /* OR of VacuumOption flags */
	Relation *RangeVar `json:"relation"` /* single table to process, or NULL */
	VaCols   List      `json:"va_cols"`  /* list of column names, or NIL for all */
}

func (node VacuumStmt) MarshalJSON() ([]byte, error) {
	type VacuumStmtMarshalAlias VacuumStmt
	return json.Marshal(map[string]interface{}{
		"VacuumStmt": (*VacuumStmtMarshalAlias)(&node),
	})
}

func (node *VacuumStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["options"] != nil {
		err = json.Unmarshal(fields["options"], &node.Options)
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

	if fields["va_cols"] != nil {
		node.VaCols.Items, err = UnmarshalNodeArrayJSON(fields["va_cols"])
		if err != nil {
			return
		}
	}

	return
}
