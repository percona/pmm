// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type DropSubscriptionStmt struct {
	Subname   *string      `json:"subname"`    /* Name of of the subscription */
	MissingOk bool         `json:"missing_ok"` /* Skip error if missing? */
	Behavior  DropBehavior `json:"behavior"`   /* RESTRICT or CASCADE behavior */
}

func (node DropSubscriptionStmt) MarshalJSON() ([]byte, error) {
	type DropSubscriptionStmtMarshalAlias DropSubscriptionStmt
	return json.Marshal(map[string]interface{}{
		"DropSubscriptionStmt": (*DropSubscriptionStmtMarshalAlias)(&node),
	})
}

func (node *DropSubscriptionStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["subname"] != nil {
		err = json.Unmarshal(fields["subname"], &node.Subname)
		if err != nil {
			return
		}
	}

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	if fields["behavior"] != nil {
		err = json.Unmarshal(fields["behavior"], &node.Behavior)
		if err != nil {
			return
		}
	}

	return
}
