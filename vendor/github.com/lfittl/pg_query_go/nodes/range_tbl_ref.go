// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeTblRef - reference to an entry in the query's rangetable
 *
 * We could use direct pointers to the RT entries and skip having these
 * nodes, but multiple pointers to the same node in a querytree cause
 * lots of headaches, so it seems better to store an index into the RT.
 */
type RangeTblRef struct {
	Rtindex int `json:"rtindex"`
}

func (node RangeTblRef) MarshalJSON() ([]byte, error) {
	type RangeTblRefMarshalAlias RangeTblRef
	return json.Marshal(map[string]interface{}{
		"RangeTblRef": (*RangeTblRefMarshalAlias)(&node),
	})
}

func (node *RangeTblRef) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["rtindex"] != nil {
		err = json.Unmarshal(fields["rtindex"], &node.Rtindex)
		if err != nil {
			return
		}
	}

	return
}
