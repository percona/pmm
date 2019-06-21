// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TriggerTransition -
 *	   representation of transition row or table naming clause
 *
 * Only transition tables are initially supported in the syntax, and only for
 * AFTER triggers, but other permutations are accepted by the parser so we can
 * give a meaningful message from C code.
 */
type TriggerTransition struct {
	Name    *string `json:"name"`
	IsNew   bool    `json:"isNew"`
	IsTable bool    `json:"isTable"`
}

func (node TriggerTransition) MarshalJSON() ([]byte, error) {
	type TriggerTransitionMarshalAlias TriggerTransition
	return json.Marshal(map[string]interface{}{
		"TriggerTransition": (*TriggerTransitionMarshalAlias)(&node),
	})
}

func (node *TriggerTransition) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["isNew"] != nil {
		err = json.Unmarshal(fields["isNew"], &node.IsNew)
		if err != nil {
			return
		}
	}

	if fields["isTable"] != nil {
		err = json.Unmarshal(fields["isTable"], &node.IsTable)
		if err != nil {
			return
		}
	}

	return
}
