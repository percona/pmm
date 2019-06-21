// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create {Aggregate|Operator|Type} Statement
 * ----------------------
 */
type DefineStmt struct {
	Kind        ObjectType `json:"kind"`          /* aggregate, operator, type */
	Oldstyle    bool       `json:"oldstyle"`      /* hack to signal old CREATE AGG syntax */
	Defnames    List       `json:"defnames"`      /* qualified name (list of Value strings) */
	Args        List       `json:"args"`          /* a list of TypeName (if needed) */
	Definition  List       `json:"definition"`    /* a list of DefElem */
	IfNotExists bool       `json:"if_not_exists"` /* just do nothing if it already exists? */
}

func (node DefineStmt) MarshalJSON() ([]byte, error) {
	type DefineStmtMarshalAlias DefineStmt
	return json.Marshal(map[string]interface{}{
		"DefineStmt": (*DefineStmtMarshalAlias)(&node),
	})
}

func (node *DefineStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["oldstyle"] != nil {
		err = json.Unmarshal(fields["oldstyle"], &node.Oldstyle)
		if err != nil {
			return
		}
	}

	if fields["defnames"] != nil {
		node.Defnames.Items, err = UnmarshalNodeArrayJSON(fields["defnames"])
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

	if fields["definition"] != nil {
		node.Definition.Items, err = UnmarshalNodeArrayJSON(fields["definition"])
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

	return
}
