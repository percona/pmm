// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type PartitionRangeDatum struct {
	Kind  PartitionRangeDatumKind `json:"kind"`
	Value Node                    `json:"value"` /* Const (or A_Const in raw tree), if kind is
	 * PARTITION_RANGE_DATUM_VALUE, else NULL */

	Location int `json:"location"` /* token location, or -1 if unknown */
}

func (node PartitionRangeDatum) MarshalJSON() ([]byte, error) {
	type PartitionRangeDatumMarshalAlias PartitionRangeDatum
	return json.Marshal(map[string]interface{}{
		"PartitionRangeDatum": (*PartitionRangeDatumMarshalAlias)(&node),
	})
}

func (node *PartitionRangeDatum) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["kind"] != nil {
		err = json.Unmarshal(fields["kind"], &node.Kind)
		if err != nil {
			return
		}
	}

	if fields["value"] != nil {
		node.Value, err = UnmarshalNodeJSON(fields["value"])
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
