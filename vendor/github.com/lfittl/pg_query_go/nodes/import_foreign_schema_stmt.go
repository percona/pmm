// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type ImportForeignSchemaStmt struct {
	ServerName   *string                 `json:"server_name"`   /* FDW server name */
	RemoteSchema *string                 `json:"remote_schema"` /* remote schema name to query */
	LocalSchema  *string                 `json:"local_schema"`  /* local schema to create objects in */
	ListType     ImportForeignSchemaType `json:"list_type"`     /* type of table list */
	TableList    List                    `json:"table_list"`    /* List of RangeVar */
	Options      List                    `json:"options"`       /* list of options to pass to FDW */
}

func (node ImportForeignSchemaStmt) MarshalJSON() ([]byte, error) {
	type ImportForeignSchemaStmtMarshalAlias ImportForeignSchemaStmt
	return json.Marshal(map[string]interface{}{
		"ImportForeignSchemaStmt": (*ImportForeignSchemaStmtMarshalAlias)(&node),
	})
}

func (node *ImportForeignSchemaStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["server_name"] != nil {
		err = json.Unmarshal(fields["server_name"], &node.ServerName)
		if err != nil {
			return
		}
	}

	if fields["remote_schema"] != nil {
		err = json.Unmarshal(fields["remote_schema"], &node.RemoteSchema)
		if err != nil {
			return
		}
	}

	if fields["local_schema"] != nil {
		err = json.Unmarshal(fields["local_schema"], &node.LocalSchema)
		if err != nil {
			return
		}
	}

	if fields["list_type"] != nil {
		err = json.Unmarshal(fields["list_type"], &node.ListType)
		if err != nil {
			return
		}
	}

	if fields["table_list"] != nil {
		node.TableList.Items, err = UnmarshalNodeArrayJSON(fields["table_list"])
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
