// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Alter Extension Statements
 * ----------------------
 */
type CreateExtensionStmt struct {
	Extname     *string `json:"extname"`
	IfNotExists bool    `json:"if_not_exists"` /* just do nothing if it already exists? */
	Options     List    `json:"options"`       /* List of DefElem nodes */
}

func (node CreateExtensionStmt) MarshalJSON() ([]byte, error) {
	type CreateExtensionStmtMarshalAlias CreateExtensionStmt
	return json.Marshal(map[string]interface{}{
		"CreateExtensionStmt": (*CreateExtensionStmtMarshalAlias)(&node),
	})
}

func (node *CreateExtensionStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["extname"] != nil {
		err = json.Unmarshal(fields["extname"], &node.Extname)
		if err != nil {
			return
		}
	}

	if fields["if_not_exists"] != nil {
		err = json.Unmarshal(fields["if_not_exists"], &node.IfNotExists)
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
