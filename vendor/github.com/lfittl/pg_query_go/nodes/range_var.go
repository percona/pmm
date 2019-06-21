// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeVar - range variable, used in FROM clauses
 *
 * Also used to represent table names in utility statements; there, the alias
 * field is not used, and inh tells whether to apply the operation
 * recursively to child tables.  In some contexts it is also useful to carry
 * a TEMP table indication here.
 */
type RangeVar struct {
	Catalogname *string `json:"catalogname"` /* the catalog (database) name, or NULL */
	Schemaname  *string `json:"schemaname"`  /* the schema name, or NULL */
	Relname     *string `json:"relname"`     /* the relation/sequence name */
	Inh         bool    `json:"inh"`         /* expand rel by inheritance? recursively act
	 * on children? */

	Relpersistence byte   `json:"relpersistence"` /* see RELPERSISTENCE_* in pg_class.h */
	Alias          *Alias `json:"alias"`          /* table alias & optional column aliases */
	Location       int    `json:"location"`       /* token location, or -1 if unknown */
}

func (node RangeVar) MarshalJSON() ([]byte, error) {
	type RangeVarMarshalAlias RangeVar
	return json.Marshal(map[string]interface{}{
		"RangeVar": (*RangeVarMarshalAlias)(&node),
	})
}

func (node *RangeVar) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["catalogname"] != nil {
		err = json.Unmarshal(fields["catalogname"], &node.Catalogname)
		if err != nil {
			return
		}
	}

	if fields["schemaname"] != nil {
		err = json.Unmarshal(fields["schemaname"], &node.Schemaname)
		if err != nil {
			return
		}
	}

	if fields["relname"] != nil {
		err = json.Unmarshal(fields["relname"], &node.Relname)
		if err != nil {
			return
		}
	}

	if fields["inh"] != nil {
		err = json.Unmarshal(fields["inh"], &node.Inh)
		if err != nil {
			return
		}
	}

	if fields["relpersistence"] != nil {
		var strVal string
		err = json.Unmarshal(fields["relpersistence"], &strVal)
		node.Relpersistence = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["alias"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["alias"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(Alias)
			node.Alias = &val
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
