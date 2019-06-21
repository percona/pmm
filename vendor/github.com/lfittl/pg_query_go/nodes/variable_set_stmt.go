// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type VariableSetStmt struct {
	Kind    VariableSetKind `json:"kind"`
	Name    *string         `json:"name"`     /* variable to be set */
	Args    List            `json:"args"`     /* List of A_Const nodes */
	IsLocal bool            `json:"is_local"` /* SET LOCAL? */
}

func (node VariableSetStmt) MarshalJSON() ([]byte, error) {
	type VariableSetStmtMarshalAlias VariableSetStmt
	return json.Marshal(map[string]interface{}{
		"VariableSetStmt": (*VariableSetStmtMarshalAlias)(&node),
	})
}

func (node *VariableSetStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["is_local"] != nil {
		err = json.Unmarshal(fields["is_local"], &node.IsLocal)
		if err != nil {
			return
		}
	}

	return
}
