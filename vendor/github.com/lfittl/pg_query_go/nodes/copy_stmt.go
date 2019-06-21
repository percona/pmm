// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Copy Statement
 *
 * We support "COPY relation FROM file", "COPY relation TO file", and
 * "COPY (query) TO file".  In any given CopyStmt, exactly one of "relation"
 * and "query" must be non-NULL.
 * ----------------------
 */
type CopyStmt struct {
	Relation *RangeVar `json:"relation"` /* the relation to copy */
	Query    Node      `json:"query"`    /* the query (SELECT or DML statement with
	 * RETURNING) to copy, as a raw parse tree */

	Attlist List `json:"attlist"` /* List of column names (as Strings), or NIL
	 * for all columns */

	IsFrom    bool    `json:"is_from"`    /* TO or FROM */
	IsProgram bool    `json:"is_program"` /* is 'filename' a program to popen? */
	Filename  *string `json:"filename"`   /* filename, or NULL for STDIN/STDOUT */
	Options   List    `json:"options"`    /* List of DefElem nodes */
}

func (node CopyStmt) MarshalJSON() ([]byte, error) {
	type CopyStmtMarshalAlias CopyStmt
	return json.Marshal(map[string]interface{}{
		"CopyStmt": (*CopyStmtMarshalAlias)(&node),
	})
}

func (node *CopyStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["query"] != nil {
		node.Query, err = UnmarshalNodeJSON(fields["query"])
		if err != nil {
			return
		}
	}

	if fields["attlist"] != nil {
		node.Attlist.Items, err = UnmarshalNodeArrayJSON(fields["attlist"])
		if err != nil {
			return
		}
	}

	if fields["is_from"] != nil {
		err = json.Unmarshal(fields["is_from"], &node.IsFrom)
		if err != nil {
			return
		}
	}

	if fields["is_program"] != nil {
		err = json.Unmarshal(fields["is_program"], &node.IsProgram)
		if err != nil {
			return
		}
	}

	if fields["filename"] != nil {
		err = json.Unmarshal(fields["filename"], &node.Filename)
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
