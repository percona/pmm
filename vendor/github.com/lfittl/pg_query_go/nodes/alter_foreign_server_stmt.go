// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Alter FOREIGN SERVER Statements
 * ----------------------
 */
type AlterForeignServerStmt struct {
	Servername *string `json:"servername"`  /* server name */
	Version    *string `json:"version"`     /* optional server version */
	Options    List    `json:"options"`     /* generic options to server */
	HasVersion bool    `json:"has_version"` /* version specified */
}

func (node AlterForeignServerStmt) MarshalJSON() ([]byte, error) {
	type AlterForeignServerStmtMarshalAlias AlterForeignServerStmt
	return json.Marshal(map[string]interface{}{
		"AlterForeignServerStmt": (*AlterForeignServerStmtMarshalAlias)(&node),
	})
}

func (node *AlterForeignServerStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["version"] != nil {
		err = json.Unmarshal(fields["version"], &node.Version)
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

	if fields["has_version"] != nil {
		err = json.Unmarshal(fields["has_version"], &node.HasVersion)
		if err != nil {
			return
		}
	}

	return
}
