// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Operator Class Statement
 * ----------------------
 */
type CreateOpClassItem struct {
	Itemtype    int             `json:"itemtype"`     /* see codes above */
	Name        *ObjectWithArgs `json:"name"`         /* operator or function name and args */
	Number      int             `json:"number"`       /* strategy num or support proc num */
	OrderFamily List            `json:"order_family"` /* only used for ordering operators */
	ClassArgs   List            `json:"class_args"`   /* amproclefttype/amprocrighttype or
	 * amoplefttype/amoprighttype */

	/* fields used for a storagetype item: */
	Storedtype *TypeName `json:"storedtype"` /* datatype stored in index */
}

func (node CreateOpClassItem) MarshalJSON() ([]byte, error) {
	type CreateOpClassItemMarshalAlias CreateOpClassItem
	return json.Marshal(map[string]interface{}{
		"CreateOpClassItem": (*CreateOpClassItemMarshalAlias)(&node),
	})
}

func (node *CreateOpClassItem) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["itemtype"] != nil {
		err = json.Unmarshal(fields["itemtype"], &node.Itemtype)
		if err != nil {
			return
		}
	}

	if fields["name"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["name"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(ObjectWithArgs)
			node.Name = &val
		}
	}

	if fields["number"] != nil {
		err = json.Unmarshal(fields["number"], &node.Number)
		if err != nil {
			return
		}
	}

	if fields["order_family"] != nil {
		node.OrderFamily.Items, err = UnmarshalNodeArrayJSON(fields["order_family"])
		if err != nil {
			return
		}
	}

	if fields["class_args"] != nil {
		node.ClassArgs.Items, err = UnmarshalNodeArrayJSON(fields["class_args"])
		if err != nil {
			return
		}
	}

	if fields["storedtype"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["storedtype"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.Storedtype = &val
		}
	}

	return
}
