// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * PartitionBoundSpec - a partition bound specification
 *
 * This represents the portion of the partition key space assigned to a
 * particular partition.  These are stored on disk in pg_class.relpartbound.
 */
type PartitionBoundSpec struct {
	Strategy byte `json:"strategy"` /* see PARTITION_STRATEGY codes above */

	/* Partitioning info for LIST strategy: */
	Listdatums List `json:"listdatums"` /* List of Consts (or A_Consts in raw tree) */

	/* Partitioning info for RANGE strategy: */
	Lowerdatums List `json:"lowerdatums"` /* List of PartitionRangeDatums */
	Upperdatums List `json:"upperdatums"` /* List of PartitionRangeDatums */

	Location int `json:"location"` /* token location, or -1 if unknown */
}

func (node PartitionBoundSpec) MarshalJSON() ([]byte, error) {
	type PartitionBoundSpecMarshalAlias PartitionBoundSpec
	return json.Marshal(map[string]interface{}{
		"PartitionBoundSpec": (*PartitionBoundSpecMarshalAlias)(&node),
	})
}

func (node *PartitionBoundSpec) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["strategy"] != nil {
		var strVal string
		err = json.Unmarshal(fields["strategy"], &strVal)
		node.Strategy = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["listdatums"] != nil {
		node.Listdatums.Items, err = UnmarshalNodeArrayJSON(fields["listdatums"])
		if err != nil {
			return
		}
	}

	if fields["lowerdatums"] != nil {
		node.Lowerdatums.Items, err = UnmarshalNodeArrayJSON(fields["lowerdatums"])
		if err != nil {
			return
		}
	}

	if fields["upperdatums"] != nil {
		node.Upperdatums.Items, err = UnmarshalNodeArrayJSON(fields["upperdatums"])
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
