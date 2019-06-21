// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Alias -
 *	  specifies an alias for a range variable; the alias might also
 *	  specify renaming of columns within the table.
 *
 * Note: colnames is a list of Value nodes (always strings).  In Alias structs
 * associated with RTEs, there may be entries corresponding to dropped
 * columns; these are normally empty strings ("").  See parsenodes.h for info.
 */
type Alias struct {
	Aliasname *string `json:"aliasname"` /* aliased rel name (never qualified) */
	Colnames  List    `json:"colnames"`  /* optional list of column aliases */
}

func (node Alias) MarshalJSON() ([]byte, error) {
	type AliasMarshalAlias Alias
	return json.Marshal(map[string]interface{}{
		"Alias": (*AliasMarshalAlias)(&node),
	})
}

func (node *Alias) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["aliasname"] != nil {
		err = json.Unmarshal(fields["aliasname"], &node.Aliasname)
		if err != nil {
			return
		}
	}

	if fields["colnames"] != nil {
		node.Colnames.Items, err = UnmarshalNodeArrayJSON(fields["colnames"])
		if err != nil {
			return
		}
	}

	return
}
