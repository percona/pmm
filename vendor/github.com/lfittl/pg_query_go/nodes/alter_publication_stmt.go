// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type AlterPublicationStmt struct {
	Pubname *string `json:"pubname"` /* Name of of the publication */

	/* parameters used for ALTER PUBLICATION ... WITH */
	Options List `json:"options"` /* List of DefElem nodes */

	/* parameters used for ALTER PUBLICATION ... ADD/DROP TABLE */
	Tables       List          `json:"tables"`         /* List of tables to add/drop */
	ForAllTables bool          `json:"for_all_tables"` /* Special publication for all tables in db */
	TableAction  DefElemAction `json:"tableAction"`    /* What action to perform with the tables */
}

func (node AlterPublicationStmt) MarshalJSON() ([]byte, error) {
	type AlterPublicationStmtMarshalAlias AlterPublicationStmt
	return json.Marshal(map[string]interface{}{
		"AlterPublicationStmt": (*AlterPublicationStmtMarshalAlias)(&node),
	})
}

func (node *AlterPublicationStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["tableAction"] != nil {
		err = json.Unmarshal(fields["tableAction"], &node.TableAction)
		if err != nil {
			return
		}
	}

	return
}
