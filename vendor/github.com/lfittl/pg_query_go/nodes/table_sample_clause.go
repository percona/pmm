// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TableSampleClause - TABLESAMPLE appearing in a transformed FROM clause
 *
 * Unlike RangeTableSample, this is a subnode of the relevant RangeTblEntry.
 */
type TableSampleClause struct {
	Tsmhandler Oid  `json:"tsmhandler"` /* OID of the tablesample handler function */
	Args       List `json:"args"`       /* tablesample argument expression(s) */
	Repeatable Node `json:"repeatable"` /* REPEATABLE expression, or NULL if none */
}

func (node TableSampleClause) MarshalJSON() ([]byte, error) {
	type TableSampleClauseMarshalAlias TableSampleClause
	return json.Marshal(map[string]interface{}{
		"TableSampleClause": (*TableSampleClauseMarshalAlias)(&node),
	})
}

func (node *TableSampleClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["tsmhandler"] != nil {
		err = json.Unmarshal(fields["tsmhandler"], &node.Tsmhandler)
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["repeatable"] != nil {
		node.Repeatable, err = UnmarshalNodeJSON(fields["repeatable"])
		if err != nil {
			return
		}
	}

	return
}
