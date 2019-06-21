// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TS Dictionary stmts: DefineStmt, RenameStmt and DropStmt are default
 */
type AlterTSDictionaryStmt struct {
	Dictname List `json:"dictname"` /* qualified name (list of Value strings) */
	Options  List `json:"options"`  /* List of DefElem nodes */
}

func (node AlterTSDictionaryStmt) MarshalJSON() ([]byte, error) {
	type AlterTSDictionaryStmtMarshalAlias AlterTSDictionaryStmt
	return json.Marshal(map[string]interface{}{
		"AlterTSDictionaryStmt": (*AlterTSDictionaryStmtMarshalAlias)(&node),
	})
}

func (node *AlterTSDictionaryStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["dictname"] != nil {
		node.Dictname.Items, err = UnmarshalNodeArrayJSON(fields["dictname"])
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
