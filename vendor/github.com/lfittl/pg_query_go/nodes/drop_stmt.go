// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Drop Table|Sequence|View|Index|Type|Domain|Conversion|Schema Statement
 * ----------------------
 */
type DropStmt struct {
	Objects    List         `json:"objects"`    /* list of names */
	RemoveType ObjectType   `json:"removeType"` /* object type */
	Behavior   DropBehavior `json:"behavior"`   /* RESTRICT or CASCADE behavior */
	MissingOk  bool         `json:"missing_ok"` /* skip error if object is missing? */
	Concurrent bool         `json:"concurrent"` /* drop index concurrently? */
}

func (node DropStmt) MarshalJSON() ([]byte, error) {
	type DropStmtMarshalAlias DropStmt
	return json.Marshal(map[string]interface{}{
		"DropStmt": (*DropStmtMarshalAlias)(&node),
	})
}

func (node *DropStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["objects"] != nil {
		node.Objects.Items, err = UnmarshalNodeArrayJSON(fields["objects"])
		if err != nil {
			return
		}
	}

	if fields["removeType"] != nil {
		err = json.Unmarshal(fields["removeType"], &node.RemoveType)
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

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	if fields["concurrent"] != nil {
		err = json.Unmarshal(fields["concurrent"], &node.Concurrent)
		if err != nil {
			return
		}
	}

	return
}
