// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * SortGroupClause -
 *		representation of ORDER BY, GROUP BY, PARTITION BY,
 *		DISTINCT, DISTINCT ON items
 *
 * You might think that ORDER BY is only interested in defining ordering,
 * and GROUP/DISTINCT are only interested in defining equality.  However,
 * one way to implement grouping is to sort and then apply a "uniq"-like
 * filter.  So it's also interesting to keep track of possible sort operators
 * for GROUP/DISTINCT, and in particular to try to sort for the grouping
 * in a way that will also yield a requested ORDER BY ordering.  So we need
 * to be able to compare ORDER BY and GROUP/DISTINCT lists, which motivates
 * the decision to give them the same representation.
 *
 * tleSortGroupRef must match ressortgroupref of exactly one entry of the
 *		query's targetlist; that is the expression to be sorted or grouped by.
 * eqop is the OID of the equality operator.
 * sortop is the OID of the ordering operator (a "<" or ">" operator),
 *		or InvalidOid if not available.
 * nulls_first means about what you'd expect.  If sortop is InvalidOid
 *		then nulls_first is meaningless and should be set to false.
 * hashable is TRUE if eqop is hashable (note this condition also depends
 *		on the datatype of the input expression).
 *
 * In an ORDER BY item, all fields must be valid.  (The eqop isn't essential
 * here, but it's cheap to get it along with the sortop, and requiring it
 * to be valid eases comparisons to grouping items.)  Note that this isn't
 * actually enough information to determine an ordering: if the sortop is
 * collation-sensitive, a collation OID is needed too.  We don't store the
 * collation in SortGroupClause because it's not available at the time the
 * parser builds the SortGroupClause; instead, consult the exposed collation
 * of the referenced targetlist expression to find out what it is.
 *
 * In a grouping item, eqop must be valid.  If the eqop is a btree equality
 * operator, then sortop should be set to a compatible ordering operator.
 * We prefer to set eqop/sortop/nulls_first to match any ORDER BY item that
 * the query presents for the same tlist item.  If there is none, we just
 * use the default ordering op for the datatype.
 *
 * If the tlist item's type has a hash opclass but no btree opclass, then
 * we will set eqop to the hash equality operator, sortop to InvalidOid,
 * and nulls_first to false.  A grouping item of this kind can only be
 * implemented by hashing, and of course it'll never match an ORDER BY item.
 *
 * The hashable flag is provided since we generally have the requisite
 * information readily available when the SortGroupClause is constructed,
 * and it's relatively expensive to get it again later.  Note there is no
 * need for a "sortable" flag since OidIsValid(sortop) serves the purpose.
 *
 * A query might have both ORDER BY and DISTINCT (or DISTINCT ON) clauses.
 * In SELECT DISTINCT, the distinctClause list is as long or longer than the
 * sortClause list, while in SELECT DISTINCT ON it's typically shorter.
 * The two lists must match up to the end of the shorter one --- the parser
 * rearranges the distinctClause if necessary to make this true.  (This
 * restriction ensures that only one sort step is needed to both satisfy the
 * ORDER BY and set up for the Unique step.  This is semantically necessary
 * for DISTINCT ON, and presents no real drawback for DISTINCT.)
 */
type SortGroupClause struct {
	TleSortGroupRef Index `json:"tleSortGroupRef"` /* reference into targetlist */
	Eqop            Oid   `json:"eqop"`            /* the equality operator ('=' op) */
	Sortop          Oid   `json:"sortop"`          /* the ordering operator ('<' op), or 0 */
	NullsFirst      bool  `json:"nulls_first"`     /* do NULLs come before normal values? */
	Hashable        bool  `json:"hashable"`        /* can eqop be implemented by hashing? */
}

func (node SortGroupClause) MarshalJSON() ([]byte, error) {
	type SortGroupClauseMarshalAlias SortGroupClause
	return json.Marshal(map[string]interface{}{
		"SortGroupClause": (*SortGroupClauseMarshalAlias)(&node),
	})
}

func (node *SortGroupClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["tleSortGroupRef"] != nil {
		err = json.Unmarshal(fields["tleSortGroupRef"], &node.TleSortGroupRef)
		if err != nil {
			return
		}
	}

	if fields["eqop"] != nil {
		err = json.Unmarshal(fields["eqop"], &node.Eqop)
		if err != nil {
			return
		}
	}

	if fields["sortop"] != nil {
		err = json.Unmarshal(fields["sortop"], &node.Sortop)
		if err != nil {
			return
		}
	}

	if fields["nulls_first"] != nil {
		err = json.Unmarshal(fields["nulls_first"], &node.NullsFirst)
		if err != nil {
			return
		}
	}

	if fields["hashable"] != nil {
		err = json.Unmarshal(fields["hashable"], &node.Hashable)
		if err != nil {
			return
		}
	}

	return
}
