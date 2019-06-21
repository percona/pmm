// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		CREATE TABLE AS Statement (a/k/a SELECT INTO)
 *
 * A query written as CREATE TABLE AS will produce this node type natively.
 * A query written as SELECT ... INTO will be transformed to this form during
 * parse analysis.
 * A query written as CREATE MATERIALIZED view will produce this node type,
 * during parse analysis, since it needs all the same data.
 *
 * The "query" field is handled similarly to EXPLAIN, though note that it
 * can be a SELECT or an EXECUTE, but not other DML statements.
 * ----------------------
 */
type CreateTableAsStmt struct {
	Query        Node        `json:"query"`          /* the query (see comments above) */
	Into         *IntoClause `json:"into"`           /* destination table */
	Relkind      ObjectType  `json:"relkind"`        /* OBJECT_TABLE or OBJECT_MATVIEW */
	IsSelectInto bool        `json:"is_select_into"` /* it was written as SELECT INTO */
	IfNotExists  bool        `json:"if_not_exists"`  /* just do nothing if it already exists? */
}

func (node CreateTableAsStmt) MarshalJSON() ([]byte, error) {
	type CreateTableAsStmtMarshalAlias CreateTableAsStmt
	return json.Marshal(map[string]interface{}{
		"CreateTableAsStmt": (*CreateTableAsStmtMarshalAlias)(&node),
	})
}

func (node *CreateTableAsStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["query"] != nil {
		node.Query, err = UnmarshalNodeJSON(fields["query"])
		if err != nil {
			return
		}
	}

	if fields["into"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["into"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(IntoClause)
			node.Into = &val
		}
	}

	if fields["relkind"] != nil {
		err = json.Unmarshal(fields["relkind"], &node.Relkind)
		if err != nil {
			return
		}
	}

	if fields["is_select_into"] != nil {
		err = json.Unmarshal(fields["is_select_into"], &node.IsSelectInto)
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
