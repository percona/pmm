// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------
 * FromExpr - represents a FROM ... WHERE ... construct
 *
 * This is both more flexible than a JoinExpr (it can have any number of
 * children, including zero) and less so --- we don't need to deal with
 * aliases and so on.  The output column set is implicitly just the union
 * of the outputs of the children.
 *----------
 */
type FromExpr struct {
	Fromlist List `json:"fromlist"` /* List of join subtrees */
	Quals    Node `json:"quals"`    /* qualifiers on join, if any */
}

func (node FromExpr) MarshalJSON() ([]byte, error) {
	type FromExprMarshalAlias FromExpr
	return json.Marshal(map[string]interface{}{
		"FromExpr": (*FromExprMarshalAlias)(&node),
	})
}

func (node *FromExpr) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["fromlist"] != nil {
		node.Fromlist.Items, err = UnmarshalNodeArrayJSON(fields["fromlist"])
		if err != nil {
			return
		}
	}

	if fields["quals"] != nil {
		node.Quals, err = UnmarshalNodeJSON(fields["quals"])
		if err != nil {
			return
		}
	}

	return
}
