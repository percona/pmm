// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node SubPlan) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("SubPlan")

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

	if node.FirstColCollation != 0 {
		ctx.WriteString("firstColCollation")
		ctx.WriteString(strconv.Itoa(int(node.FirstColCollation)))
	}

	if node.FirstColType != 0 {
		ctx.WriteString("firstColType")
		ctx.WriteString(strconv.Itoa(int(node.FirstColType)))
	}

	if node.FirstColTypmod != 0 {
		ctx.WriteString("firstColTypmod")
		ctx.WriteString(strconv.Itoa(int(node.FirstColTypmod)))
	}

	if len(node.ParParam.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ParParam.Fingerprint(&subCtx, node, "ParParam")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("parParam")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.ParallelSafe {
		ctx.WriteString("parallel_safe")
		ctx.WriteString(strconv.FormatBool(node.ParallelSafe))
	}

	if len(node.ParamIds.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ParamIds.Fingerprint(&subCtx, node, "ParamIds")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("paramIds")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	ctx.WriteString("per_call_cost")
	ctx.WriteString(strconv.FormatFloat(float64(node.PerCallCost), 'E', -1, 64))

	if node.PlanId != 0 {
		ctx.WriteString("plan_id")
		ctx.WriteString(strconv.Itoa(int(node.PlanId)))
	}

	if node.PlanName != nil {
		ctx.WriteString("plan_name")
		ctx.WriteString(*node.PlanName)
	}

	if len(node.SetParam.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.SetParam.Fingerprint(&subCtx, node, "SetParam")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("setParam")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	ctx.WriteString("startup_cost")
	ctx.WriteString(strconv.FormatFloat(float64(node.StartupCost), 'E', -1, 64))

	if int(node.SubLinkType) != 0 {
		ctx.WriteString("subLinkType")
		ctx.WriteString(strconv.Itoa(int(node.SubLinkType)))
	}

	if node.Testexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Testexpr.Fingerprint(&subCtx, node, "Testexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("testexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.UnknownEqFalse {
		ctx.WriteString("unknownEqFalse")
		ctx.WriteString(strconv.FormatBool(node.UnknownEqFalse))
	}

	if node.UseHashTable {
		ctx.WriteString("useHashTable")
		ctx.WriteString(strconv.FormatBool(node.UseHashTable))
	}

	if node.Xpr != nil {
		subCtx := FingerprintSubContext{}
		node.Xpr.Fingerprint(&subCtx, node, "Xpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("xpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
