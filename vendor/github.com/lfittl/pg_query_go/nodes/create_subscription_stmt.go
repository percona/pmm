// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type CreateSubscriptionStmt struct {
	Subname     *string `json:"subname"`     /* Name of of the subscription */
	Conninfo    *string `json:"conninfo"`    /* Connection string to publisher */
	Publication List    `json:"publication"` /* One or more publication to subscribe to */
	Options     List    `json:"options"`     /* List of DefElem nodes */
}

func (node CreateSubscriptionStmt) MarshalJSON() ([]byte, error) {
	type CreateSubscriptionStmtMarshalAlias CreateSubscriptionStmt
	return json.Marshal(map[string]interface{}{
		"CreateSubscriptionStmt": (*CreateSubscriptionStmtMarshalAlias)(&node),
	})
}

func (node *CreateSubscriptionStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["conninfo"] != nil {
		err = json.Unmarshal(fields["conninfo"], &node.Conninfo)
		if err != nil {
			return
		}
	}

	if fields["publication"] != nil {
		node.Publication.Items, err = UnmarshalNodeArrayJSON(fields["publication"])
		if err != nil {
			return
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	return
}
