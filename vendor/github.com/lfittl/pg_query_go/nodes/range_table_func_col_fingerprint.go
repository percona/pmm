// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeTableFuncCol) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeTableFuncCol")

	if node.Coldefexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Coldefexpr.Fingerprint(&subCtx, node, "Coldefexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coldefexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Colexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Colexpr.Fingerprint(&subCtx, node, "Colexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Colname != nil {
		ctx.WriteString("colname")
		ctx.WriteString(*node.Colname)
	}

	if node.ForOrdinality {
		ctx.WriteString("for_ordinality")
		ctx.WriteString(strconv.FormatBool(node.ForOrdinality))
	}

	if node.IsNotNull {
		ctx.WriteString("is_not_null")
		ctx.WriteString(strconv.FormatBool(node.IsNotNull))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if node.TypeName != nil {
		subCtx := FingerprintSubContext{}
		node.TypeName.Fingerprint(&subCtx, node, "TypeName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("typeName")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
