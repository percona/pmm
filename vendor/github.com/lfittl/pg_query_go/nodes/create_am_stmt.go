// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------------------
 *		Create ACCESS METHOD Statement
 *----------------------
 */
type CreateAmStmt struct {
	Amname      *string `json:"amname"`       /* access method name */
	HandlerName List    `json:"handler_name"` /* handler function name */
	Amtype      byte    `json:"amtype"`       /* type of access method */
}

func (node CreateAmStmt) MarshalJSON() ([]byte, error) {
	type CreateAmStmtMarshalAlias CreateAmStmt
	return json.Marshal(map[string]interface{}{
		"CreateAmStmt": (*CreateAmStmtMarshalAlias)(&node),
	})
}

func (node *CreateAmStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["amname"] != nil {
		err = json.Unmarshal(fields["amname"], &node.Amname)
		if err != nil {
			return
		}
	}

	if fields["handler_name"] != nil {
		node.HandlerName.Items, err = UnmarshalNodeArrayJSON(fields["handler_name"])
		if err != nil {
			return
		}
	}

	if fields["amtype"] != nil {
		var strVal string
		err = json.Unmarshal(fields["amtype"], &strVal)
		node.Amtype = strVal[0]
		if err != nil {
			return
		}
	}

	return
}
