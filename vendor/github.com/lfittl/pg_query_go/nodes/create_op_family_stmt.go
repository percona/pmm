// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Operator Family Statement
 * ----------------------
 */
type CreateOpFamilyStmt struct {
	Opfamilyname List    `json:"opfamilyname"` /* qualified name (list of Value strings) */
	Amname       *string `json:"amname"`       /* name of index AM opfamily is for */
}

func (node CreateOpFamilyStmt) MarshalJSON() ([]byte, error) {
	type CreateOpFamilyStmtMarshalAlias CreateOpFamilyStmt
	return json.Marshal(map[string]interface{}{
		"CreateOpFamilyStmt": (*CreateOpFamilyStmtMarshalAlias)(&node),
	})
}

func (node *CreateOpFamilyStmt) UnmarshalJSON(input []byte) (err error) {
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

	return
}
