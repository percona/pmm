// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * FieldStore
 *
 * FieldStore represents the operation of modifying one field in a tuple
 * value, yielding a new tuple value (the input is not touched!).  Like
 * the assign case of ArrayRef, this is used to implement UPDATE of a
 * portion of a column.
 *
 * A single FieldStore can actually represent updates of several different
 * fields.  The parser only generates FieldStores with single-element lists,
 * but the planner will collapse multiple updates of the same base column
 * into one FieldStore.
 * ----------------
 */
type FieldStore struct {
	Xpr        Node `json:"xpr"`
	Arg        Node `json:"arg"`        /* input tuple value */
	Newvals    List `json:"newvals"`    /* new value(s) for field(s) */
	Fieldnums  List `json:"fieldnums"`  /* integer list of field attnums */
	Resulttype Oid  `json:"resulttype"` /* type of result (same as type of arg) */

	/* Like RowExpr, we deliberately omit a typmod and collation here */
}

func (node FieldStore) MarshalJSON() ([]byte, error) {
	type FieldStoreMarshalAlias FieldStore
	return json.Marshal(map[string]interface{}{
		"FieldStore": (*FieldStoreMarshalAlias)(&node),
	})
}

func (node *FieldStore) UnmarshalJSON(input []byte) (err error) {
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

	if fields["arg"] != nil {
		node.Arg, err = UnmarshalNodeJSON(fields["arg"])
		if err != nil {
			return
		}
	}

	if fields["newvals"] != nil {
		node.Newvals.Items, err = UnmarshalNodeArrayJSON(fields["newvals"])
		if err != nil {
			return
		}
	}

	if fields["fieldnums"] != nil {
		node.Fieldnums.Items, err = UnmarshalNodeArrayJSON(fields["fieldnums"])
		if err != nil {
			return
		}
	}

	if fields["resulttype"] != nil {
		err = json.Unmarshal(fields["resulttype"], &node.Resulttype)
		if err != nil {
			return
		}
	}

	return
}
