// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		CREATE CONVERSION Statement
 * ----------------------
 */
type CreateConversionStmt struct {
	ConversionName  List    `json:"conversion_name"`   /* Name of the conversion */
	ForEncodingName *string `json:"for_encoding_name"` /* source encoding name */
	ToEncodingName  *string `json:"to_encoding_name"`  /* destination encoding name */
	FuncName        List    `json:"func_name"`         /* qualified conversion function name */
	Def             bool    `json:"def"`               /* is this a default conversion? */
}

func (node CreateConversionStmt) MarshalJSON() ([]byte, error) {
	type CreateConversionStmtMarshalAlias CreateConversionStmt
	return json.Marshal(map[string]interface{}{
		"CreateConversionStmt": (*CreateConversionStmtMarshalAlias)(&node),
	})
}

func (node *CreateConversionStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["conversion_name"] != nil {
		node.ConversionName.Items, err = UnmarshalNodeArrayJSON(fields["conversion_name"])
		if err != nil {
			return
		}
	}

	if fields["for_encoding_name"] != nil {
		err = json.Unmarshal(fields["for_encoding_name"], &node.ForEncodingName)
		if err != nil {
			return
		}
	}

	if fields["to_encoding_name"] != nil {
		err = json.Unmarshal(fields["to_encoding_name"], &node.ToEncodingName)
		if err != nil {
			return
		}
	}

	if fields["func_name"] != nil {
		node.FuncName.Items, err = UnmarshalNodeArrayJSON(fields["func_name"])
		if err != nil {
			return
		}
	}

	if fields["def"] != nil {
		err = json.Unmarshal(fields["def"], &node.Def)
		if err != nil {
			return
		}
	}

	return
}
