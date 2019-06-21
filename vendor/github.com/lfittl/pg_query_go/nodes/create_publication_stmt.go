// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type CreatePublicationStmt struct {
	Pubname      *string `json:"pubname"`        /* Name of of the publication */
	Options      List    `json:"options"`        /* List of DefElem nodes */
	Tables       List    `json:"tables"`         /* Optional list of tables to add */
	ForAllTables bool    `json:"for_all_tables"` /* Special publication for all tables in db */
}

func (node CreatePublicationStmt) MarshalJSON() ([]byte, error) {
	type CreatePublicationStmtMarshalAlias CreatePublicationStmt
	return json.Marshal(map[string]interface{}{
		"CreatePublicationStmt": (*CreatePublicationStmtMarshalAlias)(&node),
	})
}

func (node *CreatePublicationStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["pubname"] != nil {
		err = json.Unmarshal(fields["pubname"], &node.Pubname)
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

	if fields["tables"] != nil {
		node.Tables.Items, err = UnmarshalNodeArrayJSON(fields["tables"])
		if err != nil {
			return
		}
	}

	if fields["for_all_tables"] != nil {
		err = json.Unmarshal(fields["for_all_tables"], &node.ForAllTables)
		if err != nil {
			return
		}
	}

	return
}
