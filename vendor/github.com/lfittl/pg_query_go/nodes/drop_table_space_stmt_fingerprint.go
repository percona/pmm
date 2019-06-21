// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DropTableSpaceStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DropTableSpaceStmt")

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}

	if node.Tablespacename != nil {
		ctx.WriteString("tablespacename")
		ctx.WriteString(*node.Tablespacename)
	}
}
