// Auto-generated - DO NOT EDIT

package pg_query

func (node NotifyStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("NotifyStmt")

	if node.Conditionname != nil {
		ctx.WriteString("conditionname")
		ctx.WriteString(*node.Conditionname)
	}

	if node.Payload != nil {
		ctx.WriteString("payload")
		ctx.WriteString(*node.Payload)
	}
}
