// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * PartitionCmd - info for ALTER TABLE ATTACH/DETACH PARTITION commands
 */
type PartitionCmd struct {
	Name  *RangeVar           `json:"name"`  /* name of partition to attach/detach */
	Bound *PartitionBoundSpec `json:"bound"` /* FOR VALUES, if attaching */
}

func (node PartitionCmd) MarshalJSON() ([]byte, error) {
	type PartitionCmdMarshalAlias PartitionCmd
	return json.Marshal(map[string]interface{}{
		"PartitionCmd": (*PartitionCmdMarshalAlias)(&node),
	})
}

func (node *PartitionCmd) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["name"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Name = &val
		}
	}

	if fields["bound"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["bound"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(PartitionBoundSpec)
			node.Bound = &val
		}
	}

	return
}
