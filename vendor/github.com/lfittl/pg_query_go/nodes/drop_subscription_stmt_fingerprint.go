// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DropSubscriptionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DropSubscriptionStmt")

	if int(node.Behavior) != 0 {
		ctx.WriteString("behavior")
		ctx.WriteString(strconv.Itoa(int(node.Behavior)))
	}

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}

	if node.Subname != nil {
		ctx.WriteString("subname")
		ctx.WriteString(*node.Subname)
	}
}
