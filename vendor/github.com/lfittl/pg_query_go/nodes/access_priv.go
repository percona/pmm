// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * An access privilege, with optional list of column names
 * priv_name == NULL denotes ALL PRIVILEGES (only used with a column list)
 * cols == NIL denotes "all columns"
 * Note that simple "ALL PRIVILEGES" is represented as a NIL list, not
 * an AccessPriv with both fields null.
 */
type AccessPriv struct {
	PrivName *string `json:"priv_name"` /* string name of privilege */
	Cols     List    `json:"cols"`      /* list of Value strings */
}

func (node AccessPriv) MarshalJSON() ([]byte, error) {
	type AccessPrivMarshalAlias AccessPriv
	return json.Marshal(map[string]interface{}{
		"AccessPriv": (*AccessPrivMarshalAlias)(&node),
	})
}

func (node *AccessPriv) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["priv_name"] != nil {
		err = json.Unmarshal(fields["priv_name"], &node.PrivName)
		if err != nil {
			return
		}
	}

	if fields["cols"] != nil {
		node.Cols.Items, err = UnmarshalNodeArrayJSON(fields["cols"])
		if err != nil {
			return
		}
	}

	return
}
