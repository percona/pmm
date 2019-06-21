// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		DO Statement
 *
 * DoStmt is the raw parser output, InlineCodeBlock is the execution-time API
 * ----------------------
 */
type InlineCodeBlock struct {
	SourceText    *string `json:"source_text"`   /* source text of anonymous code block */
	LangOid       Oid     `json:"langOid"`       /* OID of selected language */
	LangIsTrusted bool    `json:"langIsTrusted"` /* trusted property of the language */
}

func (node InlineCodeBlock) MarshalJSON() ([]byte, error) {
	type InlineCodeBlockMarshalAlias InlineCodeBlock
	return json.Marshal(map[string]interface{}{
		"InlineCodeBlock": (*InlineCodeBlockMarshalAlias)(&node),
	})
}

func (node *InlineCodeBlock) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["source_text"] != nil {
		err = json.Unmarshal(fields["source_text"], &node.SourceText)
		if err != nil {
			return
		}
	}

	if fields["langOid"] != nil {
		err = json.Unmarshal(fields["langOid"], &node.LangOid)
		if err != nil {
			return
		}
	}

	if fields["langIsTrusted"] != nil {
		err = json.Unmarshal(fields["langIsTrusted"], &node.LangIsTrusted)
		if err != nil {
			return
		}
	}

	return
}
