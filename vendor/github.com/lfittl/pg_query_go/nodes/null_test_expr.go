// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * NullTest
 *
 * NullTest represents the operation of testing a value for NULLness.
 * The appropriate test is performed and returned as a boolean Datum.
 *
 * When argisrow is false, this simply represents a test for the null value.
 *
 * When argisrow is true, the input expression must yield a rowtype, and
 * the node implements "row IS [NOT] NULL" per the SQL standard.  This
 * includes checking individual fields for NULLness when the row datum
 * itself isn't NULL.
 *
 * NOTE: the combination of a rowtype input and argisrow==false does NOT
 * correspond to the SQL notation "row IS [NOT] NULL"; instead, this case
 * represents the SQL notation "row IS [NOT] DISTINCT FROM NULL".
 * ----------------
 */
type NullTest struct {
	Xpr          Node         `json:"xpr"`
	Arg          Node         `json:"arg"`          /* input expression */
	Nulltesttype NullTestType `json:"nulltesttype"` /* IS NULL, IS NOT NULL */
	Argisrow     bool         `json:"argisrow"`     /* T to perform field-by-field null checks */
	Location     int          `json:"location"`     /* token location, or -1 if unknown */
}

func (node NullTest) MarshalJSON() ([]byte, error) {
	type NullTestMarshalAlias NullTest
	return json.Marshal(map[string]interface{}{
		"NullTest": (*NullTestMarshalAlias)(&node),
	})
}

func (node *NullTest) UnmarshalJSON(input []byte) (err error) {
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

	if fields["nulltesttype"] != nil {
		err = json.Unmarshal(fields["nulltesttype"], &node.Nulltesttype)
		if err != nil {
			return
		}
	}

	if fields["argisrow"] != nil {
		err = json.Unmarshal(fields["argisrow"], &node.Argisrow)
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
