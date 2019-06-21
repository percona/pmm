// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TriggerTransition) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TriggerTransition")

	if node.IsNew {
		ctx.WriteString("isNew")
		ctx.WriteString(strconv.FormatBool(node.IsNew))
	}

	if node.IsTable {
		ctx.WriteString("isTable")
		ctx.WriteString(strconv.FormatBool(node.IsTable))
	}

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}
}
