// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RowMarkClause -
 *	   parser output representation of FOR [KEY] UPDATE/SHARE clauses
 *
 * Query.rowMarks contains a separate RowMarkClause node for each relation
 * identified as a FOR [KEY] UPDATE/SHARE target.  If one of these clauses
 * is applied to a subquery, we generate RowMarkClauses for all normal and
 * subquery rels in the subquery, but they are marked pushedDown = true to
 * distinguish them from clauses that were explicitly written at this query
 * level.  Also, Query.hasForUpdate tells whether there were explicit FOR
 * UPDATE/SHARE/KEY SHARE clauses in the current query level.
 */
type RowMarkClause struct {
	Rti        Index              `json:"rti"` /* range table index of target relation */
	Strength   LockClauseStrength `json:"strength"`
	WaitPolicy LockWaitPolicy     `json:"waitPolicy"` /* NOWAIT and SKIP LOCKED */
	PushedDown bool               `json:"pushedDown"` /* pushed down from higher query level? */
}

func (node RowMarkClause) MarshalJSON() ([]byte, error) {
	type RowMarkClauseMarshalAlias RowMarkClause
	return json.Marshal(map[string]interface{}{
		"RowMarkClause": (*RowMarkClauseMarshalAlias)(&node),
	})
}

func (node *RowMarkClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["rti"] != nil {
		err = json.Unmarshal(fields["rti"], &node.Rti)
		if err != nil {
			return
		}
	}

	if fields["strength"] != nil {
		err = json.Unmarshal(fields["strength"], &node.Strength)
		if err != nil {
			return
		}
	}

	if fields["waitPolicy"] != nil {
		err = json.Unmarshal(fields["waitPolicy"], &node.WaitPolicy)
		if err != nil {
			return
		}
	}

	if fields["pushedDown"] != nil {
		err = json.Unmarshal(fields["pushedDown"], &node.PushedDown)
		if err != nil {
			return
		}
	}

	return
}
