// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * ResTarget -
 *	  result target (used in target list of pre-transformed parse trees)
 *
 * In a SELECT target list, 'name' is the column label from an
 * 'AS ColumnLabel' clause, or NULL if there was none, and 'val' is the
 * value expression itself.  The 'indirection' field is not used.
 *
 * INSERT uses ResTarget in its target-column-names list.  Here, 'name' is
 * the name of the destination column, 'indirection' stores any subscripts
 * attached to the destination, and 'val' is not used.
 *
 * In an UPDATE target list, 'name' is the name of the destination column,
 * 'indirection' stores any subscripts attached to the destination, and
 * 'val' is the expression to assign.
 *
 * See A_Indirection for more info about what can appear in 'indirection'.
 */
type ResTarget struct {
	Name        *string `json:"name"`        /* column name or NULL */
	Indirection List    `json:"indirection"` /* subscripts, field names, and '*', or NIL */
	Val         Node    `json:"val"`         /* the value expression to compute or assign */
	Location    int     `json:"location"`    /* token location, or -1 if unknown */
}

func (node ResTarget) MarshalJSON() ([]byte, error) {
	type ResTargetMarshalAlias ResTarget
	return json.Marshal(map[string]interface{}{
		"ResTarget": (*ResTargetMarshalAlias)(&node),
	})
}

func (node *ResTarget) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["indirection"] != nil {
		node.Indirection.Items, err = UnmarshalNodeArrayJSON(fields["indirection"])
		if err != nil {
			return
		}
	}

	if fields["val"] != nil {
		node.Val, err = UnmarshalNodeJSON(fields["val"])
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
