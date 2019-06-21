// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *				SECURITY LABEL Statement
 * ----------------------
 */
type SecLabelStmt struct {
	Objtype  ObjectType `json:"objtype"`  /* Object's type */
	Object   Node       `json:"object"`   /* Qualified name of the object */
	Provider *string    `json:"provider"` /* Label provider (or NULL) */
	Label    *string    `json:"label"`    /* New security label to be assigned */
}

func (node SecLabelStmt) MarshalJSON() ([]byte, error) {
	type SecLabelStmtMarshalAlias SecLabelStmt
	return json.Marshal(map[string]interface{}{
		"SecLabelStmt": (*SecLabelStmtMarshalAlias)(&node),
	})
}

func (node *SecLabelStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["provider"] != nil {
		err = json.Unmarshal(fields["provider"], &node.Provider)
		if err != nil {
			return
		}
	}

	if fields["label"] != nil {
		err = json.Unmarshal(fields["label"], &node.Label)
		if err != nil {
			return
		}
	}

	return
}
