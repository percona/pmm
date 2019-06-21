// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node InlineCodeBlock) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("InlineCodeBlock")

	if node.LangIsTrusted {
		ctx.WriteString("langIsTrusted")
		ctx.WriteString(strconv.FormatBool(node.LangIsTrusted))
	}

	if node.LangOid != 0 {
		ctx.WriteString("langOid")
		ctx.WriteString(strconv.Itoa(int(node.LangOid)))
	}

	if node.SourceText != nil {
		ctx.WriteString("source_text")
		ctx.WriteString(*node.SourceText)
	}
}
