// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * PartitionElem - parse-time representation of a single partition key
 *
 * expr can be either a raw expression tree or a parse-analyzed expression.
 * We don't store these on-disk, though.
 */
type PartitionElem struct {
	Name      *string `json:"name"`      /* name of column to partition on, or NULL */
	Expr      Node    `json:"expr"`      /* expression to partition on, or NULL */
	Collation List    `json:"collation"` /* name of collation; NIL = default */
	Opclass   List    `json:"opclass"`   /* name of desired opclass; NIL = default */
	Location  int     `json:"location"`  /* token location, or -1 if unknown */
}

func (node PartitionElem) MarshalJSON() ([]byte, error) {
	type PartitionElemMarshalAlias PartitionElem
	return json.Marshal(map[string]interface{}{
		"PartitionElem": (*PartitionElemMarshalAlias)(&node),
	})
}

func (node *PartitionElem) UnmarshalJSON(input []byte) (err error) {
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

	if fields["expr"] != nil {
		node.Expr, err = UnmarshalNodeJSON(fields["expr"])
		if err != nil {
			return
		}
	}

	if fields["collation"] != nil {
		node.Collation.Items, err = UnmarshalNodeArrayJSON(fields["collation"])
		if err != nil {
			return
		}
	}

	if fields["opclass"] != nil {
		node.Opclass.Items, err = UnmarshalNodeArrayJSON(fields["opclass"])
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
