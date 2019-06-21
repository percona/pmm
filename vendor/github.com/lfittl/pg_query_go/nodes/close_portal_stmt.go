// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Close Portal Statement
 * ----------------------
 */
type ClosePortalStmt struct {
	Portalname *string `json:"portalname"` /* name of the portal (cursor) */

	/* NULL means CLOSE ALL */
}

func (node ClosePortalStmt) MarshalJSON() ([]byte, error) {
	type ClosePortalStmtMarshalAlias ClosePortalStmt
	return json.Marshal(map[string]interface{}{
		"ClosePortalStmt": (*ClosePortalStmtMarshalAlias)(&node),
	})
}

func (node *ClosePortalStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["portalname"] != nil {
		err = json.Unmarshal(fields["portalname"], &node.Portalname)
		if err != nil {
			return
		}
	}

	return
}
