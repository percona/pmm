// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * NextValueExpr - get next value from sequence
 *
 * This has the same effect as calling the nextval() function, but it does not
 * check permissions on the sequence.  This is used for identity columns,
 * where the sequence is an implicit dependency without its own permissions.
 */
type NextValueExpr struct {
	Xpr    Node `json:"xpr"`
	Seqid  Oid  `json:"seqid"`
	TypeId Oid  `json:"typeId"`
}

func (node NextValueExpr) MarshalJSON() ([]byte, error) {
	type NextValueExprMarshalAlias NextValueExpr
	return json.Marshal(map[string]interface{}{
		"NextValueExpr": (*NextValueExprMarshalAlias)(&node),
	})
}

func (node *NextValueExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["seqid"] != nil {
		err = json.Unmarshal(fields["seqid"], &node.Seqid)
		if err != nil {
			return
		}
	}

	if fields["typeId"] != nil {
		err = json.Unmarshal(fields["typeId"], &node.TypeId)
		if err != nil {
			return
		}
	}

	return
}
