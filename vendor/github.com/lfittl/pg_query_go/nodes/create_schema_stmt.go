// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Schema Statement
 *
 * NOTE: the schemaElts list contains raw parsetrees for component statements
 * of the schema, such as CREATE TABLE, GRANT, etc.  These are analyzed and
 * executed after the schema itself is created.
 * ----------------------
 */
type CreateSchemaStmt struct {
	Schemaname  *string   `json:"schemaname"`    /* the name of the schema to create */
	Authrole    *RoleSpec `json:"authrole"`      /* the owner of the created schema */
	SchemaElts  List      `json:"schemaElts"`    /* schema components (list of parsenodes) */
	IfNotExists bool      `json:"if_not_exists"` /* just do nothing if schema already exists? */
}

func (node CreateSchemaStmt) MarshalJSON() ([]byte, error) {
	type CreateSchemaStmtMarshalAlias CreateSchemaStmt
	return json.Marshal(map[string]interface{}{
		"CreateSchemaStmt": (*CreateSchemaStmtMarshalAlias)(&node),
	})
}

func (node *CreateSchemaStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["schemaname"] != nil {
		err = json.Unmarshal(fields["schemaname"], &node.Schemaname)
		if err != nil {
			return
		}
	}

	if fields["authrole"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["authrole"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RoleSpec)
			node.Authrole = &val
		}
	}

	if fields["schemaElts"] != nil {
		node.SchemaElts.Items, err = UnmarshalNodeArrayJSON(fields["schemaElts"])
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
