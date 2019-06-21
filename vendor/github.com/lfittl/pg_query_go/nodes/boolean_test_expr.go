// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * BooleanTest
 *
 * BooleanTest represents the operation of determining whether a boolean
 * is TRUE, FALSE, or UNKNOWN (ie, NULL).  All six meaningful combinations
 * are supported.  Note that a NULL input does *not* cause a NULL result.
 * The appropriate test is performed and returned as a boolean Datum.
 */
type BooleanTest struct {
	Xpr          Node         `json:"xpr"`
	Arg          Node         `json:"arg"`          /* input expression */
	Booltesttype BoolTestType `json:"booltesttype"` /* test type */
	Location     int          `json:"location"`     /* token location, or -1 if unknown */
}

func (node BooleanTest) MarshalJSON() ([]byte, error) {
	type BooleanTestMarshalAlias BooleanTest
	return json.Marshal(map[string]interface{}{
		"BooleanTest": (*BooleanTestMarshalAlias)(&node),
	})
}

func (node *BooleanTest) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xpr"] != nil {
		node.Xpr, err = UnmarshalNodeJSON(fields["xpr"])
		if err != nil {
			return
		}
	}

	if fields["arg"] != nil {
		node.Arg, err = UnmarshalNodeJSON(fields["arg"])
		if err != nil {
			return
		}
	}

	if fields["booltesttype"] != nil {
		err = json.Unmarshal(fields["booltesttype"], &node.Booltesttype)
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
