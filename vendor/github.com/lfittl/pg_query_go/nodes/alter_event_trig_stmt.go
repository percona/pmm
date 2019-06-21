// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Alter EVENT TRIGGER Statement
 * ----------------------
 */
type AlterEventTrigStmt struct {
	Trigname  *string `json:"trigname"`  /* TRIGGER's name */
	Tgenabled byte    `json:"tgenabled"` /* trigger's firing configuration WRT
	 * session_replication_role */
}

func (node AlterEventTrigStmt) MarshalJSON() ([]byte, error) {
	type AlterEventTrigStmtMarshalAlias AlterEventTrigStmt
	return json.Marshal(map[string]interface{}{
		"AlterEventTrigStmt": (*AlterEventTrigStmtMarshalAlias)(&node),
	})
}

func (node *AlterEventTrigStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["tgenabled"] != nil {
		var strVal string
		err = json.Unmarshal(fields["tgenabled"], &strVal)
		node.Tgenabled = strVal[0]
		if err != nil {
			return
		}
	}

	return
}
