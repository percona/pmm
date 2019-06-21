// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Domain Statement
 * ----------------------
 */
type CreateDomainStmt struct {
	Domainname  List           `json:"domainname"`  /* qualified name (list of Value strings) */
	TypeName    *TypeName      `json:"typeName"`    /* the base type */
	CollClause  *CollateClause `json:"collClause"`  /* untransformed COLLATE spec, if any */
	Constraints List           `json:"constraints"` /* constraints (list of Constraint nodes) */
}

func (node CreateDomainStmt) MarshalJSON() ([]byte, error) {
	type CreateDomainStmtMarshalAlias CreateDomainStmt
	return json.Marshal(map[string]interface{}{
		"CreateDomainStmt": (*CreateDomainStmtMarshalAlias)(&node),
	})
}

func (node *CreateDomainStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["domainname"] != nil {
		node.Domainname.Items, err = UnmarshalNodeArrayJSON(fields["domainname"])
		if err != nil {
			return
		}
	}

	if fields["typeName"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["typeName"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.TypeName = &val
		}
	}

	if fields["collClause"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["collClause"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(CollateClause)
			node.CollClause = &val
		}
	}

	if fields["constraints"] != nil {
		node.Constraints.Items, err = UnmarshalNodeArrayJSON(fields["constraints"])
		if err != nil {
			return
		}
	}

	return
}
