// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		SET CONSTRAINTS Statement
 * ----------------------
 */
type ConstraintsSetStmt struct {
	Constraints List `json:"constraints"` /* List of names as RangeVars */
	Deferred    bool `json:"deferred"`
}

func (node ConstraintsSetStmt) MarshalJSON() ([]byte, error) {
	type ConstraintsSetStmtMarshalAlias ConstraintsSetStmt
	return json.Marshal(map[string]interface{}{
		"ConstraintsSetStmt": (*ConstraintsSetStmtMarshalAlias)(&node),
	})
}

func (node *ConstraintsSetStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["constraints"] != nil {
		node.Constraints.Items, err = UnmarshalNodeArrayJSON(fields["constraints"])
		if err != nil {
			return
		}
	}

	if fields["deferred"] != nil {
		err = json.Unmarshal(fields["deferred"], &node.Deferred)
		if err != nil {
			return
		}
	}

	return
}
