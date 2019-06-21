// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * OnConflictClause -
 *		representation of ON CONFLICT clause
 *
 * Note: OnConflictClause does not propagate into the Query representation.
 */
type OnConflictClause struct {
	Action      OnConflictAction `json:"action"`      /* DO NOTHING or UPDATE? */
	Infer       *InferClause     `json:"infer"`       /* Optional index inference clause */
	TargetList  List             `json:"targetList"`  /* the target list (of ResTarget) */
	WhereClause Node             `json:"whereClause"` /* qualifications */
	Location    int              `json:"location"`    /* token location, or -1 if unknown */
}

func (node OnConflictClause) MarshalJSON() ([]byte, error) {
	type OnConflictClauseMarshalAlias OnConflictClause
	return json.Marshal(map[string]interface{}{
		"OnConflictClause": (*OnConflictClauseMarshalAlias)(&node),
	})
}

func (node *OnConflictClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["action"] != nil {
		err = json.Unmarshal(fields["action"], &node.Action)
		if err != nil {
			return
		}
	}

	if fields["infer"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["infer"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(InferClause)
			node.Infer = &val
		}
	}

	if fields["targetList"] != nil {
		node.TargetList.Items, err = UnmarshalNodeArrayJSON(fields["targetList"])
		if err != nil {
			return
		}
	}

	if fields["whereClause"] != nil {
		node.WhereClause, err = UnmarshalNodeJSON(fields["whereClause"])
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
