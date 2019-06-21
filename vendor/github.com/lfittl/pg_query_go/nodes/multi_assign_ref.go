// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * MultiAssignRef - element of a row source expression for UPDATE
 *
 * In an UPDATE target list, when we have SET (a,b,c) = row-valued-expression,
 * we generate separate ResTarget items for each of a,b,c.  Their "val" trees
 * are MultiAssignRef nodes numbered 1..n, linking to a common copy of the
 * row-valued-expression (which parse analysis will process only once, when
 * handling the MultiAssignRef with colno=1).
 */
type MultiAssignRef struct {
	Source   Node `json:"source"`   /* the row-valued expression */
	Colno    int  `json:"colno"`    /* column number for this target (1..n) */
	Ncolumns int  `json:"ncolumns"` /* number of targets in the construct */
}

func (node MultiAssignRef) MarshalJSON() ([]byte, error) {
	type MultiAssignRefMarshalAlias MultiAssignRef
	return json.Marshal(map[string]interface{}{
		"MultiAssignRef": (*MultiAssignRefMarshalAlias)(&node),
	})
}

func (node *MultiAssignRef) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["source"] != nil {
		node.Source, err = UnmarshalNodeJSON(fields["source"])
		if err != nil {
			return
		}
	}

	if fields["colno"] != nil {
		err = json.Unmarshal(fields["colno"], &node.Colno)
		if err != nil {
			return
		}
	}

	if fields["ncolumns"] != nil {
		err = json.Unmarshal(fields["ncolumns"], &node.Ncolumns)
		if err != nil {
			return
		}
	}

	return
}
