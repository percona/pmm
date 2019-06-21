// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DropdbStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DropdbStmt")

	if node.Dbname != nil {
		ctx.WriteString("dbname")
		ctx.WriteString(*node.Dbname)
	}

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}
}
