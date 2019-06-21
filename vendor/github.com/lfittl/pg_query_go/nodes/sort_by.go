// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * SortBy - for ORDER BY clause
 */
type SortBy struct {
	Node        Node        `json:"node"`         /* expression to sort on */
	SortbyDir   SortByDir   `json:"sortby_dir"`   /* ASC/DESC/USING/default */
	SortbyNulls SortByNulls `json:"sortby_nulls"` /* NULLS FIRST/LAST */
	UseOp       List        `json:"useOp"`        /* name of op to use, if SORTBY_USING */
	Location    int         `json:"location"`     /* operator location, or -1 if none/unknown */
}

func (node SortBy) MarshalJSON() ([]byte, error) {
	type SortByMarshalAlias SortBy
	return json.Marshal(map[string]interface{}{
		"SortBy": (*SortByMarshalAlias)(&node),
	})
}

func (node *SortBy) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["node"] != nil {
		node.Node, err = UnmarshalNodeJSON(fields["node"])
		if err != nil {
			return
		}
	}

	if fields["sortby_dir"] != nil {
		err = json.Unmarshal(fields["sortby_dir"], &node.SortbyDir)
		if err != nil {
			return
		}
	}

	if fields["sortby_nulls"] != nil {
		err = json.Unmarshal(fields["sortby_nulls"], &node.SortbyNulls)
		if err != nil {
			return
		}
	}

	if fields["useOp"] != nil {
		node.UseOp.Items, err = UnmarshalNodeArrayJSON(fields["useOp"])
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
