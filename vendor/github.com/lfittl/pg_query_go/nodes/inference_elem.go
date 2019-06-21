// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * InferenceElem - an element of a unique index inference specification
 *
 * This mostly matches the structure of IndexElems, but having a dedicated
 * primnode allows for a clean separation between the use of index parameters
 * by utility commands, and this node.
 */
type InferenceElem struct {
	Xpr          Node `json:"xpr"`
	Expr         Node `json:"expr"`         /* expression to infer from, or NULL */
	Infercollid  Oid  `json:"infercollid"`  /* OID of collation, or InvalidOid */
	Inferopclass Oid  `json:"inferopclass"` /* OID of att opclass, or InvalidOid */
}

func (node InferenceElem) MarshalJSON() ([]byte, error) {
	type InferenceElemMarshalAlias InferenceElem
	return json.Marshal(map[string]interface{}{
		"InferenceElem": (*InferenceElemMarshalAlias)(&node),
	})
}

func (node *InferenceElem) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xpr"] != nil {
		node.Xpr, err = UnmarshalNodeJSON(fields["xpr"])
		if err != nil {
			return
		}
	}

	if fields["expr"] != nil {
		node.Expr, err = UnmarshalNodeJSON(fields["expr"])
		if err != nil {
			return
		}
	}

	if fields["infercollid"] != nil {
		err = json.Unmarshal(fields["infercollid"], &node.Infercollid)
		if err != nil {
			return
		}
	}

	if fields["inferopclass"] != nil {
		err = json.Unmarshal(fields["inferopclass"], &node.Inferopclass)
		if err != nil {
			return
		}
	}

	return
}
