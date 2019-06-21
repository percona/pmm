// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node FuncCall) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("FuncCall")

	if node.AggDistinct {
		ctx.WriteString("agg_distinct")
		ctx.WriteString(strconv.FormatBool(node.AggDistinct))
	}

	if node.AggFilter != nil {
		subCtx := FingerprintSubContext{}
		node.AggFilter.Fingerprint(&subCtx, node, "AggFilter")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("agg_filter")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.AggOrder.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.AggOrder.Fingerprint(&subCtx, node, "AggOrder")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("agg_order")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.AggStar {
		ctx.WriteString("agg_star")
		ctx.WriteString(strconv.FormatBool(node.AggStar))
	}

	if node.AggWithinGroup {
		ctx.WriteString("agg_within_group")
		ctx.WriteString(strconv.FormatBool(node.AggWithinGroup))
	}

	if len(node.Args.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Args.Fingerprint(&subCtx, node, "Args")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.FuncVariadic {
		ctx.WriteString("func_variadic")
		ctx.WriteString(strconv.FormatBool(node.FuncVariadic))
	}

	if len(node.Funcname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funcname.Fingerprint(&subCtx, node, "Funcname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funcname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if node.Over != nil {
		subCtx := FingerprintSubContext{}
		node.Over.Fingerprint(&subCtx, node, "Over")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("over")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
