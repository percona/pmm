// Auto-generated - DO NOT EDIT

package pg_query

func (node BitString) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("BitString")
	if len(node.Str) > 0 {
		ctx.WriteString("str")
		ctx.WriteString(node.Str)
	}
}
