// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ParamExternData) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ParamExternData")

	if node.Isnull {
		ctx.WriteString("isnull")
		ctx.WriteString(strconv.FormatBool(node.Isnull))
	}

	if node.Pflags != 0 {
		ctx.WriteString("pflags")
		ctx.WriteString(strconv.Itoa(int(node.Pflags)))
	}

	if node.Ptype != 0 {
		ctx.WriteString("ptype")
		ctx.WriteString(strconv.Itoa(int(node.Ptype)))
	}
}
