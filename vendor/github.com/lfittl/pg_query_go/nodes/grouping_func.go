// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * GroupingFunc
 *
 * A GroupingFunc is a GROUPING(...) expression, which behaves in many ways
 * like an aggregate function (e.g. it "belongs" to a specific query level,
 * which might not be the one immediately containing it), but also differs in
 * an important respect: it never evaluates its arguments, they merely
 * designate expressions from the GROUP BY clause of the query level to which
 * it belongs.
 *
 * The spec defines the evaluation of GROUPING() purely by syntactic
 * replacement, but we make it a real expression for optimization purposes so
 * that one Agg node can handle multiple grouping sets at once.  Evaluating the
 * result only needs the column positions to check against the grouping set
 * being projected.  However, for EXPLAIN to produce meaningful output, we have
 * to keep the original expressions around, since expression deparse does not
 * give us any feasible way to get at the GROUP BY clause.
 *
 * Also, we treat two GroupingFunc nodes as equal if they have equal arguments
 * lists and agglevelsup, without comparing the refs and cols annotations.
 *
 * In raw parse output we have only the args list; parse analysis fills in the
 * refs list, and the planner fills in the cols list.
 */
type GroupingFunc struct {
	Xpr  Node `json:"xpr"`
	Args List `json:"args"` /* arguments, not evaluated but kept for
	 * benefit of EXPLAIN etc. */

	Refs        List  `json:"refs"`        /* ressortgrouprefs of arguments */
	Cols        List  `json:"cols"`        /* actual column positions set by planner */
	Agglevelsup Index `json:"agglevelsup"` /* same as Aggref.agglevelsup */
	Location    int   `json:"location"`    /* token location */
}

func (node GroupingFunc) MarshalJSON() ([]byte, error) {
	type GroupingFuncMarshalAlias GroupingFunc
	return json.Marshal(map[string]interface{}{
		"GroupingFunc": (*GroupingFuncMarshalAlias)(&node),
	})
}

func (node *GroupingFunc) UnmarshalJSON(input []byte) (err error) {
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

	if fields["refs"] != nil {
		node.Refs.Items, err = UnmarshalNodeArrayJSON(fields["refs"])
		if err != nil {
			return
		}
	}

	if fields["cols"] != nil {
		node.Cols.Items, err = UnmarshalNodeArrayJSON(fields["cols"])
		if err != nil {
			return
		}
	}

	if fields["agglevelsup"] != nil {
		err = json.Unmarshal(fields["agglevelsup"], &node.Agglevelsup)
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
