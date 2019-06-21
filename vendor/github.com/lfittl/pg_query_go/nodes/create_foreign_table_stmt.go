// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create FOREIGN TABLE Statement
 * ----------------------
 */
type CreateForeignTableStmt struct {
	Base       CreateStmt `json:"base"`
	Servername *string    `json:"servername"`
	Options    List       `json:"options"`
}

func (node CreateForeignTableStmt) MarshalJSON() ([]byte, error) {
	type CreateForeignTableStmtMarshalAlias CreateForeignTableStmt
	return json.Marshal(map[string]interface{}{
		"CreateForeignTableStmt": (*CreateForeignTableStmtMarshalAlias)(&node),
	})
}

func (node *CreateForeignTableStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["base"] != nil {
		var nodeField Node
		nodeField, err = UnmarshalNodeJSON(fields["base"])
		if err != nil {
			return
		}
		node.Base = nodeField.(CreateStmt)
	}

	if fields["servername"] != nil {
		err = json.Unmarshal(fields["servername"], &node.Servername)
		if err != nil {
			return
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	return
}
