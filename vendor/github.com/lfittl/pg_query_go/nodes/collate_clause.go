// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * CollateClause - a COLLATE expression
 */
type CollateClause struct {
	Arg      Node `json:"arg"`      /* input expression */
	Collname List `json:"collname"` /* possibly-qualified collation name */
	Location int  `json:"location"` /* token location, or -1 if unknown */
}

func (node CollateClause) MarshalJSON() ([]byte, error) {
	type CollateClauseMarshalAlias CollateClause
	return json.Marshal(map[string]interface{}{
		"CollateClause": (*CollateClauseMarshalAlias)(&node),
	})
}

func (node *CollateClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["arg"] != nil {
		node.Arg, err = UnmarshalNodeJSON(fields["arg"])
		if err != nil {
			return
		}
	}

	if fields["collname"] != nil {
		node.Collname.Items, err = UnmarshalNodeArrayJSON(fields["collname"])
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
