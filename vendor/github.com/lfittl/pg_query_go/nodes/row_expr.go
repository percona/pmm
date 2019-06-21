// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RowExpr - a ROW() expression
 *
 * Note: the list of fields must have a one-for-one correspondence with
 * physical fields of the associated rowtype, although it is okay for it
 * to be shorter than the rowtype.  That is, the N'th list element must
 * match up with the N'th physical field.  When the N'th physical field
 * is a dropped column (attisdropped) then the N'th list element can just
 * be a NULL constant.  (This case can only occur for named composite types,
 * not RECORD types, since those are built from the RowExpr itself rather
 * than vice versa.)  It is important not to assume that length(args) is
 * the same as the number of columns logically present in the rowtype.
 *
 * colnames provides field names in cases where the names can't easily be
 * obtained otherwise.  Names *must* be provided if row_typeid is RECORDOID.
 * If row_typeid identifies a known composite type, colnames can be NIL to
 * indicate the type's cataloged field names apply.  Note that colnames can
 * be non-NIL even for a composite type, and typically is when the RowExpr
 * was created by expanding a whole-row Var.  This is so that we can retain
 * the column alias names of the RTE that the Var referenced (which would
 * otherwise be very difficult to extract from the parsetree).  Like the
 * args list, colnames is one-for-one with physical fields of the rowtype.
 */
type RowExpr struct {
	Xpr       Node `json:"xpr"`
	Args      List `json:"args"`       /* the fields */
	RowTypeid Oid  `json:"row_typeid"` /* RECORDOID or a composite type's ID */

	/*
	 * Note: we deliberately do NOT store a typmod.  Although a typmod will be
	 * associated with specific RECORD types at runtime, it will differ for
	 * different backends, and so cannot safely be stored in stored
	 * parsetrees.  We must assume typmod -1 for a RowExpr node.
	 *
	 * We don't need to store a collation either.  The result type is
	 * necessarily composite, and composite types never have a collation.
	 */
	RowFormat CoercionForm `json:"row_format"` /* how to display this node */
	Colnames  List         `json:"colnames"`   /* list of String, or NIL */
	Location  int          `json:"location"`   /* token location, or -1 if unknown */
}

func (node RowExpr) MarshalJSON() ([]byte, error) {
	type RowExprMarshalAlias RowExpr
	return json.Marshal(map[string]interface{}{
		"RowExpr": (*RowExprMarshalAlias)(&node),
	})
}

func (node *RowExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["row_typeid"] != nil {
		err = json.Unmarshal(fields["row_typeid"], &node.RowTypeid)
		if err != nil {
			return
		}
	}

	if fields["row_format"] != nil {
		err = json.Unmarshal(fields["row_format"], &node.RowFormat)
		if err != nil {
			return
		}
	}

	if fields["colnames"] != nil {
		node.Colnames.Items, err = UnmarshalNodeArrayJSON(fields["colnames"])
		if err != nil {
			return
		}
	}

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
		if err != nil {
			return
		}
	}

	return
}
