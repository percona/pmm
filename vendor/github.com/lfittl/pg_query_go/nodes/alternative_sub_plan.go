// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * AlternativeSubPlan - expression node for a choice among SubPlans
 *
 * The subplans are given as a List so that the node definition need not
 * change if there's ever more than two alternatives.  For the moment,
 * though, there are always exactly two; and the first one is the fast-start
 * plan.
 */
type AlternativeSubPlan struct {
	Xpr      Node `json:"xpr"`
	Subplans List `json:"subplans"` /* SubPlan(s) with equivalent results */
}

func (node AlternativeSubPlan) MarshalJSON() ([]byte, error) {
	type AlternativeSubPlanMarshalAlias AlternativeSubPlan
	return json.Marshal(map[string]interface{}{
		"AlternativeSubPlan": (*AlternativeSubPlanMarshalAlias)(&node),
	})
}

func (node *AlternativeSubPlan) UnmarshalJSON(input []byte) (err error) {
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

	if fields["subplans"] != nil {
		node.Subplans.Items, err = UnmarshalNodeArrayJSON(fields["subplans"])
		if err != nil {
			return
		}
	}

	return
}
