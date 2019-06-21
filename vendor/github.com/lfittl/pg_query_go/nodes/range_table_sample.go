// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeTableSample - TABLESAMPLE appearing in a raw FROM clause
 *
 * This node, appearing only in raw parse trees, represents
 *		<relation> TABLESAMPLE <method> (<params>) REPEATABLE (<num>)
 * Currently, the <relation> can only be a RangeVar, but we might in future
 * allow RangeSubselect and other options.  Note that the RangeTableSample
 * is wrapped around the node representing the <relation>, rather than being
 * a subfield of it.
 */
type RangeTableSample struct {
	Relation   Node `json:"relation"`   /* relation to be sampled */
	Method     List `json:"method"`     /* sampling method name (possibly qualified) */
	Args       List `json:"args"`       /* argument(s) for sampling method */
	Repeatable Node `json:"repeatable"` /* REPEATABLE expression, or NULL if none */
	Location   int  `json:"location"`   /* method name location, or -1 if unknown */
}

func (node RangeTableSample) MarshalJSON() ([]byte, error) {
	type RangeTableSampleMarshalAlias RangeTableSample
	return json.Marshal(map[string]interface{}{
		"RangeTableSample": (*RangeTableSampleMarshalAlias)(&node),
	})
}

func (node *RangeTableSample) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["relation"] != nil {
		node.Relation, err = UnmarshalNodeJSON(fields["relation"])
		if err != nil {
			return
		}
	}

	if fields["method"] != nil {
		node.Method.Items, err = UnmarshalNodeArrayJSON(fields["method"])
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["repeatable"] != nil {
		node.Repeatable, err = UnmarshalNodeJSON(fields["repeatable"])
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
