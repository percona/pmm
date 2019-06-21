// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*--------------------
 * TargetEntry -
 *	   a target entry (used in query target lists)
 *
 * Strictly speaking, a TargetEntry isn't an expression node (since it can't
 * be evaluated by ExecEvalExpr).  But we treat it as one anyway, since in
 * very many places it's convenient to process a whole query targetlist as a
 * single expression tree.
 *
 * In a SELECT's targetlist, resno should always be equal to the item's
 * ordinal position (counting from 1).  However, in an INSERT or UPDATE
 * targetlist, resno represents the attribute number of the destination
 * column for the item; so there may be missing or out-of-order resnos.
 * It is even legal to have duplicated resnos; consider
 *		UPDATE table SET arraycol[1] = ..., arraycol[2] = ..., ...
 * The two meanings come together in the executor, because the planner
 * transforms INSERT/UPDATE tlists into a normalized form with exactly
 * one entry for each column of the destination table.  Before that's
 * happened, however, it is risky to assume that resno == position.
 * Generally get_tle_by_resno() should be used rather than list_nth()
 * to fetch tlist entries by resno, and only in SELECT should you assume
 * that resno is a unique identifier.
 *
 * resname is required to represent the correct column name in non-resjunk
 * entries of top-level SELECT targetlists, since it will be used as the
 * column title sent to the frontend.  In most other contexts it is only
 * a debugging aid, and may be wrong or even NULL.  (In particular, it may
 * be wrong in a tlist from a stored rule, if the referenced column has been
 * renamed by ALTER TABLE since the rule was made.  Also, the planner tends
 * to store NULL rather than look up a valid name for tlist entries in
 * non-toplevel plan nodes.)  In resjunk entries, resname should be either
 * a specific system-generated name (such as "ctid") or NULL; anything else
 * risks confusing ExecGetJunkAttribute!
 *
 * ressortgroupref is used in the representation of ORDER BY, GROUP BY, and
 * DISTINCT items.  Targetlist entries with ressortgroupref=0 are not
 * sort/group items.  If ressortgroupref>0, then this item is an ORDER BY,
 * GROUP BY, and/or DISTINCT target value.  No two entries in a targetlist
 * may have the same nonzero ressortgroupref --- but there is no particular
 * meaning to the nonzero values, except as tags.  (For example, one must
 * not assume that lower ressortgroupref means a more significant sort key.)
 * The order of the associated SortGroupClause lists determine the semantics.
 *
 * resorigtbl/resorigcol identify the source of the column, if it is a
 * simple reference to a column of a base table (or view).  If it is not
 * a simple reference, these fields are zeroes.
 *
 * If resjunk is true then the column is a working column (such as a sort key)
 * that should be removed from the final output of the query.  Resjunk columns
 * must have resnos that cannot duplicate any regular column's resno.  Also
 * note that there are places that assume resjunk columns come after non-junk
 * columns.
 *--------------------
 */
type TargetEntry struct {
	Xpr             Node       `json:"xpr"`
	Expr            Node       `json:"expr"`            /* expression to evaluate */
	Resno           AttrNumber `json:"resno"`           /* attribute number (see notes above) */
	Resname         *string    `json:"resname"`         /* name of the column (could be NULL) */
	Ressortgroupref Index      `json:"ressortgroupref"` /* nonzero if referenced by a sort/group
	 * clause */

	Resorigtbl Oid        `json:"resorigtbl"` /* OID of column's source table */
	Resorigcol AttrNumber `json:"resorigcol"` /* column's number in source table */
	Resjunk    bool       `json:"resjunk"`    /* set to true to eliminate the attribute from
	 * final target list */
}

func (node TargetEntry) MarshalJSON() ([]byte, error) {
	type TargetEntryMarshalAlias TargetEntry
	return json.Marshal(map[string]interface{}{
		"TargetEntry": (*TargetEntryMarshalAlias)(&node),
	})
}

func (node *TargetEntry) UnmarshalJSON(input []byte) (err error) {
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

	if fields["resno"] != nil {
		err = json.Unmarshal(fields["resno"], &node.Resno)
		if err != nil {
			return
		}
	}

	if fields["resname"] != nil {
		err = json.Unmarshal(fields["resname"], &node.Resname)
		if err != nil {
			return
		}
	}

	if fields["ressortgroupref"] != nil {
		err = json.Unmarshal(fields["ressortgroupref"], &node.Ressortgroupref)
		if err != nil {
			return
		}
	}

	if fields["resorigtbl"] != nil {
		err = json.Unmarshal(fields["resorigtbl"], &node.Resorigtbl)
		if err != nil {
			return
		}
	}

	if fields["resorigcol"] != nil {
		err = json.Unmarshal(fields["resorigcol"], &node.Resorigcol)
		if err != nil {
			return
		}
	}

	if fields["resjunk"] != nil {
		err = json.Unmarshal(fields["resjunk"], &node.Resjunk)
		if err != nil {
			return
		}
	}

	return
}
