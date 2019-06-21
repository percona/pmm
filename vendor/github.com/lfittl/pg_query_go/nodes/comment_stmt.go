// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *				Comment On Statement
 * ----------------------
 */
type CommentStmt struct {
	Objtype ObjectType `json:"objtype"` /* Object's type */
	Object  Node       `json:"object"`  /* Qualified name of the object */
	Comment *string    `json:"comment"` /* Comment to insert, or NULL to remove */
}

func (node CommentStmt) MarshalJSON() ([]byte, error) {
	type CommentStmtMarshalAlias CommentStmt
	return json.Marshal(map[string]interface{}{
		"CommentStmt": (*CommentStmtMarshalAlias)(&node),
	})
}

func (node *CommentStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["objtype"] != nil {
		err = json.Unmarshal(fields["objtype"], &node.Objtype)
		if err != nil {
			return
		}
	}

	if fields["object"] != nil {
		node.Object, err = UnmarshalNodeJSON(fields["object"])
		if err != nil {
			return
		}
	}

	if fields["comment"] != nil {
		err = json.Unmarshal(fields["comment"], &node.Comment)
		if err != nil {
			return
		}
	}

	return
}
