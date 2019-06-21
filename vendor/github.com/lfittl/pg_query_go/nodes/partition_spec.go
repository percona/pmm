// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * PartitionSpec - parse-time representation of a partition key specification
 *
 * This represents the key space we will be partitioning on.
 */
type PartitionSpec struct {
	Strategy   *string `json:"strategy"`   /* partitioning strategy ('list' or 'range') */
	PartParams List    `json:"partParams"` /* List of PartitionElems */
	Location   int     `json:"location"`   /* token location, or -1 if unknown */
}

func (node PartitionSpec) MarshalJSON() ([]byte, error) {
	type PartitionSpecMarshalAlias PartitionSpec
	return json.Marshal(map[string]interface{}{
		"PartitionSpec": (*PartitionSpecMarshalAlias)(&node),
	})
}

func (node *PartitionSpec) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["strategy"] != nil {
		err = json.Unmarshal(fields["strategy"], &node.Strategy)
		if err != nil {
			return
		}
	}

	if fields["partParams"] != nil {
		node.PartParams.Items, err = UnmarshalNodeArrayJSON(fields["partParams"])
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
