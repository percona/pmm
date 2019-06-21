// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type GroupingSet struct {
	Kind     GroupingSetKind `json:"kind"`
	Content  List            `json:"content"`
	Location int             `json:"location"`
}

func (node GroupingSet) MarshalJSON() ([]byte, error) {
	type GroupingSetMarshalAlias GroupingSet
	return json.Marshal(map[string]interface{}{
		"GroupingSet": (*GroupingSetMarshalAlias)(&node),
	})
}

func (node *GroupingSet) UnmarshalJSON(input []byte) (err error) {
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

	if fields["content"] != nil {
		node.Content.Items, err = UnmarshalNodeArrayJSON(fields["content"])
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
