// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Rule Statement
 * ----------------------
 */
type RuleStmt struct {
	Relation    *RangeVar `json:"relation"`    /* relation the rule is for */
	Rulename    *string   `json:"rulename"`    /* name of the rule */
	WhereClause Node      `json:"whereClause"` /* qualifications */
	Event       CmdType   `json:"event"`       /* SELECT, INSERT, etc */
	Instead     bool      `json:"instead"`     /* is a 'do instead'? */
	Actions     List      `json:"actions"`     /* the action statements */
	Replace     bool      `json:"replace"`     /* OR REPLACE */
}

func (node RuleStmt) MarshalJSON() ([]byte, error) {
	type RuleStmtMarshalAlias RuleStmt
	return json.Marshal(map[string]interface{}{
		"RuleStmt": (*RuleStmtMarshalAlias)(&node),
	})
}

func (node *RuleStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["relation"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["relation"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Relation = &val
		}
	}

	if fields["rulename"] != nil {
		err = json.Unmarshal(fields["rulename"], &node.Rulename)
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

	if fields["event"] != nil {
		err = json.Unmarshal(fields["event"], &node.Event)
		if err != nil {
			return
		}
	}

	if fields["instead"] != nil {
		err = json.Unmarshal(fields["instead"], &node.Instead)
		if err != nil {
			return
		}
	}

	if fields["actions"] != nil {
		node.Actions.Items, err = UnmarshalNodeArrayJSON(fields["actions"])
		if err != nil {
			return
		}
	}

	if fields["replace"] != nil {
		err = json.Unmarshal(fields["replace"], &node.Replace)
		if err != nil {
			return
		}
	}

	return
}
