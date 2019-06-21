// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * IndexElem - index parameters (used in CREATE INDEX, and in ON CONFLICT)
 *
 * For a plain index attribute, 'name' is the name of the table column to
 * index, and 'expr' is NULL.  For an index expression, 'name' is NULL and
 * 'expr' is the expression tree.
 */
type IndexElem struct {
	Name          *string     `json:"name"`           /* name of attribute to index, or NULL */
	Expr          Node        `json:"expr"`           /* expression to index, or NULL */
	Indexcolname  *string     `json:"indexcolname"`   /* name for index column; NULL = default */
	Collation     List        `json:"collation"`      /* name of collation; NIL = default */
	Opclass       List        `json:"opclass"`        /* name of desired opclass; NIL = default */
	Ordering      SortByDir   `json:"ordering"`       /* ASC/DESC/default */
	NullsOrdering SortByNulls `json:"nulls_ordering"` /* FIRST/LAST/default */
}

func (node IndexElem) MarshalJSON() ([]byte, error) {
	type IndexElemMarshalAlias IndexElem
	return json.Marshal(map[string]interface{}{
		"IndexElem": (*IndexElemMarshalAlias)(&node),
	})
}

func (node *IndexElem) UnmarshalJSON(input []byte) (err error) {
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

	if fields["indexcolname"] != nil {
		err = json.Unmarshal(fields["indexcolname"], &node.Indexcolname)
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

	if fields["ordering"] != nil {
		err = json.Unmarshal(fields["ordering"], &node.Ordering)
		if err != nil {
			return
		}
	}

	if fields["nulls_ordering"] != nil {
		err = json.Unmarshal(fields["nulls_ordering"], &node.NullsOrdering)
		if err != nil {
			return
		}
	}

	return
}
