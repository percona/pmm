// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type WithCheckOption struct {
	Kind     WCOKind `json:"kind"`     /* kind of WCO */
	Relname  *string `json:"relname"`  /* name of relation that specified the WCO */
	Polname  *string `json:"polname"`  /* name of RLS policy being checked */
	Qual     Node    `json:"qual"`     /* constraint qual to check */
	Cascaded bool    `json:"cascaded"` /* true for a cascaded WCO on a view */
}

func (node WithCheckOption) MarshalJSON() ([]byte, error) {
	type WithCheckOptionMarshalAlias WithCheckOption
	return json.Marshal(map[string]interface{}{
		"WithCheckOption": (*WithCheckOptionMarshalAlias)(&node),
	})
}

func (node *WithCheckOption) UnmarshalJSON(input []byte) (err error) {
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

	if fields["relname"] != nil {
		err = json.Unmarshal(fields["relname"], &node.Relname)
		if err != nil {
			return
		}
	}

	if fields["polname"] != nil {
		err = json.Unmarshal(fields["polname"], &node.Polname)
		if err != nil {
			return
		}
	}

	if fields["qual"] != nil {
		node.Qual, err = UnmarshalNodeJSON(fields["qual"])
		if err != nil {
			return
		}
	}

	if fields["cascaded"] != nil {
		err = json.Unmarshal(fields["cascaded"], &node.Cascaded)
		if err != nil {
			return
		}
	}

	return
}
