// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Note: ObjectWithArgs carries only the types of the input parameters of the
 * function.  So it is sufficient to identify an existing function, but it
 * is not enough info to define a function nor to call it.
 */
type ObjectWithArgs struct {
	Objname         List `json:"objname"`          /* qualified name of function/operator */
	Objargs         List `json:"objargs"`          /* list of Typename nodes */
	ArgsUnspecified bool `json:"args_unspecified"` /* argument list was omitted, so name must
	 * be unique (note that objargs == NIL
	 * means zero args) */
}

func (node ObjectWithArgs) MarshalJSON() ([]byte, error) {
	type ObjectWithArgsMarshalAlias ObjectWithArgs
	return json.Marshal(map[string]interface{}{
		"ObjectWithArgs": (*ObjectWithArgsMarshalAlias)(&node),
	})
}

func (node *ObjectWithArgs) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["objname"] != nil {
		node.Objname.Items, err = UnmarshalNodeArrayJSON(fields["objname"])
		if err != nil {
			return
		}
	}

	if fields["objargs"] != nil {
		node.Objargs.Items, err = UnmarshalNodeArrayJSON(fields["objargs"])
		if err != nil {
			return
		}
	}

	if fields["args_unspecified"] != nil {
		err = json.Unmarshal(fields["args_unspecified"], &node.ArgsUnspecified)
		if err != nil {
			return
		}
	}

	return
}
