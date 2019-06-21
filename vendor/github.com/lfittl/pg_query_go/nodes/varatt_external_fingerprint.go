// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node varatt_external) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("varatt_external")

	if node.VaExtsize != 0 {
		ctx.WriteString("va_extsize")
		ctx.WriteString(strconv.Itoa(int(node.VaExtsize)))
	}

	if node.VaRawsize != 0 {
		ctx.WriteString("va_rawsize")
		ctx.WriteString(strconv.Itoa(int(node.VaRawsize)))
	}

	if node.VaToastrelid != 0 {
		ctx.WriteString("va_toastrelid")
		ctx.WriteString(strconv.Itoa(int(node.VaToastrelid)))
	}

	if node.VaValueid != 0 {
		ctx.WriteString("va_valueid")
		ctx.WriteString(strconv.Itoa(int(node.VaValueid)))
	}
}
