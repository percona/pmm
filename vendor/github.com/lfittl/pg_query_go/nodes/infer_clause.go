// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * InferClause -
 *		ON CONFLICT unique index inference clause
 *
 * Note: InferClause does not propagate into the Query representation.
 */
type InferClause struct {
	IndexElems  List    `json:"indexElems"`  /* IndexElems to infer unique index */
	WhereClause Node    `json:"whereClause"` /* qualification (partial-index predicate) */
	Conname     *string `json:"conname"`     /* Constraint name, or NULL if unnamed */
	Location    int     `json:"location"`    /* token location, or -1 if unknown */
}

func (node InferClause) MarshalJSON() ([]byte, error) {
	type InferClauseMarshalAlias InferClause
	return json.Marshal(map[string]interface{}{
		"InferClause": (*InferClauseMarshalAlias)(&node),
	})
}

func (node *InferClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["indexElems"] != nil {
		node.IndexElems.Items, err = UnmarshalNodeArrayJSON(fields["indexElems"])
		if err != nil {
			return
		}
	}

	if fields["whereClause"] != nil {
		node.WhereClause, err = UnmarshalNodeJSON(fields["whereClause"])
		if err != nil {
			return
		}
	}

	if fields["conname"] != nil {
		err = json.Unmarshal(fields["conname"], &node.Conname)
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
