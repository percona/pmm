// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Drop PROCEDURAL LANGUAGE Statements
 *		Create PROCEDURAL LANGUAGE Statements
 * ----------------------
 */
type CreatePLangStmt struct {
	Replace     bool    `json:"replace"`     /* T => replace if already exists */
	Plname      *string `json:"plname"`      /* PL name */
	Plhandler   List    `json:"plhandler"`   /* PL call handler function (qual. name) */
	Plinline    List    `json:"plinline"`    /* optional inline function (qual. name) */
	Plvalidator List    `json:"plvalidator"` /* optional validator function (qual. name) */
	Pltrusted   bool    `json:"pltrusted"`   /* PL is trusted */
}

func (node CreatePLangStmt) MarshalJSON() ([]byte, error) {
	type CreatePLangStmtMarshalAlias CreatePLangStmt
	return json.Marshal(map[string]interface{}{
		"CreatePLangStmt": (*CreatePLangStmtMarshalAlias)(&node),
	})
}

func (node *CreatePLangStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["replace"] != nil {
		err = json.Unmarshal(fields["replace"], &node.Replace)
		if err != nil {
			return
		}
	}

	if fields["plname"] != nil {
		err = json.Unmarshal(fields["plname"], &node.Plname)
		if err != nil {
			return
		}
	}

	if fields["plhandler"] != nil {
		node.Plhandler.Items, err = UnmarshalNodeArrayJSON(fields["plhandler"])
		if err != nil {
			return
		}
	}

	if fields["plinline"] != nil {
		node.Plinline.Items, err = UnmarshalNodeArrayJSON(fields["plinline"])
		if err != nil {
			return
		}
	}

	if fields["plvalidator"] != nil {
		node.Plvalidator.Items, err = UnmarshalNodeArrayJSON(fields["plvalidator"])
		if err != nil {
			return
		}
	}

	if fields["pltrusted"] != nil {
		err = json.Unmarshal(fields["pltrusted"], &node.Pltrusted)
		if err != nil {
			return
		}
	}

	return
}
