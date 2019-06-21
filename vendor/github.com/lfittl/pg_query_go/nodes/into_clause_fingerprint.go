// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node IntoClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("IntoClause")

	if len(node.ColNames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ColNames.Fingerprint(&subCtx, node, "ColNames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colNames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.OnCommit) != 0 {
		ctx.WriteString("onCommit")
		ctx.WriteString(strconv.Itoa(int(node.OnCommit)))
	}

	if len(node.Options.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Options.Fingerprint(&subCtx, node, "Options")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("options")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Rel != nil {
		subCtx := FingerprintSubContext{}
		node.Rel.Fingerprint(&subCtx, node, "Rel")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("rel")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.SkipData {
		ctx.WriteString("skipData")
		ctx.WriteString(strconv.FormatBool(node.SkipData))
	}

	if node.TableSpaceName != nil {
		ctx.WriteString("tableSpaceName")
		ctx.WriteString(*node.TableSpaceName)
	}

	if node.ViewQuery != nil {
		subCtx := FingerprintSubContext{}
		node.ViewQuery.Fingerprint(&subCtx, node, "ViewQuery")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("viewQuery")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
