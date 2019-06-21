// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 * ALTER object DEPENDS ON EXTENSION extname
 * ----------------------
 */
type AlterObjectDependsStmt struct {
	ObjectType ObjectType `json:"objectType"` /* OBJECT_FUNCTION, OBJECT_TRIGGER, etc */
	Relation   *RangeVar  `json:"relation"`   /* in case a table is involved */
	Object     Node       `json:"object"`     /* name of the object */
	Extname    Node       `json:"extname"`    /* extension name */
}

func (node AlterObjectDependsStmt) MarshalJSON() ([]byte, error) {
	type AlterObjectDependsStmtMarshalAlias AlterObjectDependsStmt
	return json.Marshal(map[string]interface{}{
		"AlterObjectDependsStmt": (*AlterObjectDependsStmtMarshalAlias)(&node),
	})
}

func (node *AlterObjectDependsStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["objectType"] != nil {
		err = json.Unmarshal(fields["objectType"], &node.ObjectType)
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

	if fields["object"] != nil {
		node.Object, err = UnmarshalNodeJSON(fields["object"])
		if err != nil {
			return
		}
	}

	if fields["extname"] != nil {
		node.Extname, err = UnmarshalNodeJSON(fields["extname"])
		if err != nil {
			return
		}
	}

	return
}
