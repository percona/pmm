// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterTSConfigurationStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterTSConfigurationStmt")

	if len(node.Cfgname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Cfgname.Fingerprint(&subCtx, node, "Cfgname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("cfgname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Dicts.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Dicts.Fingerprint(&subCtx, node, "Dicts")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("dicts")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}

	if node.Override {
		ctx.WriteString("override")
		ctx.WriteString(strconv.FormatBool(node.Override))
	}

	if node.Replace {
		ctx.WriteString("replace")
		ctx.WriteString(strconv.FormatBool(node.Replace))
	}

	if len(node.Tokentype.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Tokentype.Fingerprint(&subCtx, node, "Tokentype")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("tokentype")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
