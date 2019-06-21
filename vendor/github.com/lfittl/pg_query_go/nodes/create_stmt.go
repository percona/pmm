// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Table Statement
 *
 * NOTE: in the raw gram.y output, ColumnDef and Constraint nodes are
 * intermixed in tableElts, and constraints is NIL.  After parse analysis,
 * tableElts contains just ColumnDefs, and constraints contains just
 * Constraint nodes (in fact, only CONSTR_CHECK nodes, in the present
 * implementation).
 * ----------------------
 */
type CreateStmt struct {
	Relation     *RangeVar `json:"relation"`     /* relation to create */
	TableElts    List      `json:"tableElts"`    /* column definitions (list of ColumnDef) */
	InhRelations List      `json:"inhRelations"` /* relations to inherit from (list of
	 * inhRelation) */

	Partbound      *PartitionBoundSpec `json:"partbound"`      /* FOR VALUES clause */
	Partspec       *PartitionSpec      `json:"partspec"`       /* PARTITION BY clause */
	OfTypename     *TypeName           `json:"ofTypename"`     /* OF typename */
	Constraints    List                `json:"constraints"`    /* constraints (list of Constraint nodes) */
	Options        List                `json:"options"`        /* options from WITH clause */
	Oncommit       OnCommitAction      `json:"oncommit"`       /* what do we do at COMMIT? */
	Tablespacename *string             `json:"tablespacename"` /* table space to use, or NULL */
	IfNotExists    bool                `json:"if_not_exists"`  /* just do nothing if it already exists? */
}

func (node CreateStmt) MarshalJSON() ([]byte, error) {
	type CreateStmtMarshalAlias CreateStmt
	return json.Marshal(map[string]interface{}{
		"CreateStmt": (*CreateStmtMarshalAlias)(&node),
	})
}

func (node *CreateStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["tableElts"] != nil {
		node.TableElts.Items, err = UnmarshalNodeArrayJSON(fields["tableElts"])
		if err != nil {
			return
		}
	}

	if fields["inhRelations"] != nil {
		node.InhRelations.Items, err = UnmarshalNodeArrayJSON(fields["inhRelations"])
		if err != nil {
			return
		}
	}

	if fields["partbound"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["partbound"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(PartitionBoundSpec)
			node.Partbound = &val
		}
	}

	if fields["partspec"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["partspec"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(PartitionSpec)
			node.Partspec = &val
		}
	}

	if fields["ofTypename"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["ofTypename"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.OfTypename = &val
		}
	}

	if fields["constraints"] != nil {
		node.Constraints.Items, err = UnmarshalNodeArrayJSON(fields["constraints"])
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

	if fields["oncommit"] != nil {
		err = json.Unmarshal(fields["oncommit"], &node.Oncommit)
		if err != nil {
			return
		}
	}

	if fields["tablespacename"] != nil {
		err = json.Unmarshal(fields["tablespacename"], &node.Tablespacename)
		if err != nil {
			return
		}
	}

	if fields["if_not_exists"] != nil {
		err = json.Unmarshal(fields["if_not_exists"], &node.IfNotExists)
		if err != nil {
			return
		}
	}

	return
}
