// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 *		RawStmt --- container for any one statement's raw parse tree
 *
 * Parse analysis converts a raw parse tree headed by a RawStmt node into
 * an analyzed statement headed by a Query node.  For optimizable statements,
 * the conversion is complex.  For utility statements, the parser usually just
 * transfers the raw parse tree (sans RawStmt) into the utilityStmt field of
 * the Query node, and all the useful work happens at execution time.
 *
 * stmt_location/stmt_len identify the portion of the source text string
 * containing this raw statement (useful for multi-statement strings).
 */
type RawStmt struct {
	Stmt         Node `json:"stmt"`          /* raw parse tree */
	StmtLocation int  `json:"stmt_location"` /* start location, or -1 if unknown */
	StmtLen      int  `json:"stmt_len"`      /* length in bytes; 0 means "rest of string" */
}

func (node RawStmt) MarshalJSON() ([]byte, error) {
	type RawStmtMarshalAlias RawStmt
	return json.Marshal(map[string]interface{}{
		"RawStmt": (*RawStmtMarshalAlias)(&node),
	})
}

func (node *RawStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["stmt"] != nil {
		node.Stmt, err = UnmarshalNodeJSON(fields["stmt"])
		if err != nil {
			return
		}
	}

	if fields["stmt_location"] != nil {
		err = json.Unmarshal(fields["stmt_location"], &node.StmtLocation)
		if err != nil {
			return
		}
	}

	if fields["stmt_len"] != nil {
		err = json.Unmarshal(fields["stmt_len"], &node.StmtLen)
		if err != nil {
			return
		}
	}

	return
}
