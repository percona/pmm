// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Statistics Statement
 * ----------------------
 */
type CreateStatsStmt struct {
	Defnames    List `json:"defnames"`      /* qualified name (list of Value strings) */
	StatTypes   List `json:"stat_types"`    /* stat types (list of Value strings) */
	Exprs       List `json:"exprs"`         /* expressions to build statistics on */
	Relations   List `json:"relations"`     /* rels to build stats on (list of RangeVar) */
	IfNotExists bool `json:"if_not_exists"` /* do nothing if stats name already exists */
}

func (node CreateStatsStmt) MarshalJSON() ([]byte, error) {
	type CreateStatsStmtMarshalAlias CreateStatsStmt
	return json.Marshal(map[string]interface{}{
		"CreateStatsStmt": (*CreateStatsStmtMarshalAlias)(&node),
	})
}

func (node *CreateStatsStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["defnames"] != nil {
		node.Defnames.Items, err = UnmarshalNodeArrayJSON(fields["defnames"])
		if err != nil {
			return
		}
	}

	if fields["stat_types"] != nil {
		node.StatTypes.Items, err = UnmarshalNodeArrayJSON(fields["stat_types"])
		if err != nil {
			return
		}
	}

	if fields["exprs"] != nil {
		node.Exprs.Items, err = UnmarshalNodeArrayJSON(fields["exprs"])
		if err != nil {
			return
		}
	}

	if fields["relations"] != nil {
		node.Relations.Items, err = UnmarshalNodeArrayJSON(fields["relations"])
		if err != nil {
			return
		}
	}

	if fields["if_not_exists"] != nil {
		err = json.Unmarshal(fields["if_not_exists"], &node.IfNotExists)
		if err != nil {
			return
		}
	}

	return
}
