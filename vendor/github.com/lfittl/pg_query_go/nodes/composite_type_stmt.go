// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Type Statement, composite types
 * ----------------------
 */
type CompositeTypeStmt struct {
	Typevar    *RangeVar `json:"typevar"`    /* the composite type to be created */
	Coldeflist List      `json:"coldeflist"` /* list of ColumnDef nodes */
}

func (node CompositeTypeStmt) MarshalJSON() ([]byte, error) {
	type CompositeTypeStmtMarshalAlias CompositeTypeStmt
	return json.Marshal(map[string]interface{}{
		"CompositeTypeStmt": (*CompositeTypeStmtMarshalAlias)(&node),
	})
}

func (node *CompositeTypeStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["typevar"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["typevar"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Typevar = &val
		}
	}

	if fields["coldeflist"] != nil {
		node.Coldeflist.Items, err = UnmarshalNodeArrayJSON(fields["coldeflist"])
		if err != nil {
			return
		}
	}

	return
}
