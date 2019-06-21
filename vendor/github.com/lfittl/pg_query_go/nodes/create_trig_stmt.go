// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create TRIGGER Statement
 * ----------------------
 */
type CreateTrigStmt struct {
	Trigname *string   `json:"trigname"` /* TRIGGER's name */
	Relation *RangeVar `json:"relation"` /* relation trigger is on */
	Funcname List      `json:"funcname"` /* qual. name of function to call */
	Args     List      `json:"args"`     /* list of (T_String) Values or NIL */
	Row      bool      `json:"row"`      /* ROW/STATEMENT */

	/* timing uses the TRIGGER_TYPE bits defined in catalog/pg_trigger.h */
	Timing int16 `json:"timing"` /* BEFORE, AFTER, or INSTEAD */

	/* events uses the TRIGGER_TYPE bits defined in catalog/pg_trigger.h */
	Events       int16 `json:"events"`       /* "OR" of INSERT/UPDATE/DELETE/TRUNCATE */
	Columns      List  `json:"columns"`      /* column names, or NIL for all columns */
	WhenClause   Node  `json:"whenClause"`   /* qual expression, or NULL if none */
	Isconstraint bool  `json:"isconstraint"` /* This is a constraint trigger */

	/* explicitly named transition data */
	TransitionRels List `json:"transitionRels"` /* TriggerTransition nodes, or NIL if none */

	/* The remaining fields are only used for constraint triggers */
	Deferrable   bool      `json:"deferrable"`   /* [NOT] DEFERRABLE */
	Initdeferred bool      `json:"initdeferred"` /* INITIALLY {DEFERRED|IMMEDIATE} */
	Constrrel    *RangeVar `json:"constrrel"`    /* opposite relation, if RI trigger */
}

func (node CreateTrigStmt) MarshalJSON() ([]byte, error) {
	type CreateTrigStmtMarshalAlias CreateTrigStmt
	return json.Marshal(map[string]interface{}{
		"CreateTrigStmt": (*CreateTrigStmtMarshalAlias)(&node),
	})
}

func (node *CreateTrigStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["trigname"] != nil {
		err = json.Unmarshal(fields["trigname"], &node.Trigname)
		if err != nil {
			return
		}
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

	if fields["funcname"] != nil {
		node.Funcname.Items, err = UnmarshalNodeArrayJSON(fields["funcname"])
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["row"] != nil {
		err = json.Unmarshal(fields["row"], &node.Row)
		if err != nil {
			return
		}
	}

	if fields["timing"] != nil {
		err = json.Unmarshal(fields["timing"], &node.Timing)
		if err != nil {
			return
		}
	}

	if fields["events"] != nil {
		err = json.Unmarshal(fields["events"], &node.Events)
		if err != nil {
			return
		}
	}

	if fields["columns"] != nil {
		node.Columns.Items, err = UnmarshalNodeArrayJSON(fields["columns"])
		if err != nil {
			return
		}
	}

	if fields["whenClause"] != nil {
		node.WhenClause, err = UnmarshalNodeJSON(fields["whenClause"])
		if err != nil {
			return
		}
	}

	if fields["isconstraint"] != nil {
		err = json.Unmarshal(fields["isconstraint"], &node.Isconstraint)
		if err != nil {
			return
		}
	}

	if fields["transitionRels"] != nil {
		node.TransitionRels.Items, err = UnmarshalNodeArrayJSON(fields["transitionRels"])
		if err != nil {
			return
		}
	}

	if fields["deferrable"] != nil {
		err = json.Unmarshal(fields["deferrable"], &node.Deferrable)
		if err != nil {
			return
		}
	}

	if fields["initdeferred"] != nil {
		err = json.Unmarshal(fields["initdeferred"], &node.Initdeferred)
		if err != nil {
			return
		}
	}

	if fields["constrrel"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["constrrel"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Constrrel = &val
		}
	}

	return
}
