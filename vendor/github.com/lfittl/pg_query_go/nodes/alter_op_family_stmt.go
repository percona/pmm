// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Alter Operator Family Statement
 * ----------------------
 */
type AlterOpFamilyStmt struct {
	Opfamilyname List    `json:"opfamilyname"` /* qualified name (list of Value strings) */
	Amname       *string `json:"amname"`       /* name of index AM opfamily is for */
	IsDrop       bool    `json:"isDrop"`       /* ADD or DROP the items? */
	Items        List    `json:"items"`        /* List of CreateOpClassItem nodes */
}

func (node AlterOpFamilyStmt) MarshalJSON() ([]byte, error) {
	type AlterOpFamilyStmtMarshalAlias AlterOpFamilyStmt
	return json.Marshal(map[string]interface{}{
		"AlterOpFamilyStmt": (*AlterOpFamilyStmtMarshalAlias)(&node),
	})
}

func (node *AlterOpFamilyStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["opfamilyname"] != nil {
		node.Opfamilyname.Items, err = UnmarshalNodeArrayJSON(fields["opfamilyname"])
		if err != nil {
			return
		}
	}

	if fields["amname"] != nil {
		err = json.Unmarshal(fields["amname"], &node.Amname)
		if err != nil {
			return
		}
	}

	if fields["isDrop"] != nil {
		err = json.Unmarshal(fields["isDrop"], &node.IsDrop)
		if err != nil {
			return
		}
	}

	if fields["items"] != nil {
		node.Items.Items, err = UnmarshalNodeArrayJSON(fields["items"])
		if err != nil {
			return
		}
	}

	return
}
