// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	Alter Domain
 *
 * The fields are used in different ways by the different variants of
 * this command.
 * ----------------------
 */
type AlterDomainStmt struct {
	Subtype byte `json:"subtype"` /*------------
	 *	T = alter column default
	 *	N = alter column drop not null
	 *	O = alter column set not null
	 *	C = add constraint
	 *	X = drop constraint
	 *------------
	 */

	TypeName  List         `json:"typeName"`   /* domain to work on */
	Name      *string      `json:"name"`       /* column or constraint name to act on */
	Def       Node         `json:"def"`        /* definition of default or constraint */
	Behavior  DropBehavior `json:"behavior"`   /* RESTRICT or CASCADE for DROP cases */
	MissingOk bool         `json:"missing_ok"` /* skip error if missing? */
}

func (node AlterDomainStmt) MarshalJSON() ([]byte, error) {
	type AlterDomainStmtMarshalAlias AlterDomainStmt
	return json.Marshal(map[string]interface{}{
		"AlterDomainStmt": (*AlterDomainStmtMarshalAlias)(&node),
	})
}

func (node *AlterDomainStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["subtype"] != nil {
		var strVal string
		err = json.Unmarshal(fields["subtype"], &strVal)
		node.Subtype = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["typeName"] != nil {
		node.TypeName.Items, err = UnmarshalNodeArrayJSON(fields["typeName"])
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

	if fields["def"] != nil {
		node.Def, err = UnmarshalNodeJSON(fields["def"])
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

	return
}
