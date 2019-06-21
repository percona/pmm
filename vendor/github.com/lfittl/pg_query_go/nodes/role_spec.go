// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type RoleSpec struct {
	Roletype RoleSpecType `json:"roletype"` /* Type of this rolespec */
	Rolename *string      `json:"rolename"` /* filled only for ROLESPEC_CSTRING */
	Location int          `json:"location"` /* token location, or -1 if unknown */
}

func (node RoleSpec) MarshalJSON() ([]byte, error) {
	type RoleSpecMarshalAlias RoleSpec
	return json.Marshal(map[string]interface{}{
		"RoleSpec": (*RoleSpecMarshalAlias)(&node),
	})
}

func (node *RoleSpec) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["roletype"] != nil {
		err = json.Unmarshal(fields["roletype"], &node.Roletype)
		if err != nil {
			return
		}
	}

	if fields["rolename"] != nil {
		err = json.Unmarshal(fields["rolename"], &node.Rolename)
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
