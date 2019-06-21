// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Alter FOREIGN SERVER Statements
 * ----------------------
 */
type CreateForeignServerStmt struct {
	Servername  *string `json:"servername"`    /* server name */
	Servertype  *string `json:"servertype"`    /* optional server type */
	Version     *string `json:"version"`       /* optional server version */
	Fdwname     *string `json:"fdwname"`       /* FDW name */
	IfNotExists bool    `json:"if_not_exists"` /* just do nothing if it already exists? */
	Options     List    `json:"options"`       /* generic options to server */
}

func (node CreateForeignServerStmt) MarshalJSON() ([]byte, error) {
	type CreateForeignServerStmtMarshalAlias CreateForeignServerStmt
	return json.Marshal(map[string]interface{}{
		"CreateForeignServerStmt": (*CreateForeignServerStmtMarshalAlias)(&node),
	})
}

func (node *CreateForeignServerStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["servername"] != nil {
		err = json.Unmarshal(fields["servername"], &node.Servername)
		if err != nil {
			return
		}
	}

	if fields["servertype"] != nil {
		err = json.Unmarshal(fields["servertype"], &node.Servertype)
		if err != nil {
			return
		}
	}

	if fields["version"] != nil {
		err = json.Unmarshal(fields["version"], &node.Version)
		if err != nil {
			return
		}
	}

	if fields["fdwname"] != nil {
		err = json.Unmarshal(fields["fdwname"], &node.Fdwname)
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
