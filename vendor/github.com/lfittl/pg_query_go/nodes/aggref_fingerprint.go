// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node Aggref) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("Aggref")

	if len(node.Aggargtypes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Aggargtypes.Fingerprint(&subCtx, node, "Aggargtypes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aggargtypes")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Aggcollid != 0 {
		ctx.WriteString("aggcollid")
		ctx.WriteString(strconv.Itoa(int(node.Aggcollid)))
	}

	if len(node.Aggdirectargs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Aggdirectargs.Fingerprint(&subCtx, node, "Aggdirectargs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aggdirectargs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Aggdistinct.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Aggdistinct.Fingerprint(&subCtx, node, "Aggdistinct")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aggdistinct")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Aggfilter != nil {
		subCtx := FingerprintSubContext{}
		node.Aggfilter.Fingerprint(&subCtx, node, "Aggfilter")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aggfilter")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Aggfnoid != 0 {
		ctx.WriteString("aggfnoid")
		ctx.WriteString(strconv.Itoa(int(node.Aggfnoid)))
	}

	if node.Aggkind != 0 {
		ctx.WriteString("aggkind")
		ctx.WriteString(string(node.Aggkind))

	}

	if node.Agglevelsup != 0 {
		ctx.WriteString("agglevelsup")
		ctx.WriteString(strconv.Itoa(int(node.Agglevelsup)))
	}

	if len(node.Aggorder.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Aggorder.Fingerprint(&subCtx, node, "Aggorder")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aggorder")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Aggsplit) != 0 {
		ctx.WriteString("aggsplit")
		ctx.WriteString(strconv.Itoa(int(node.Aggsplit)))
	}

	if node.Aggstar {
		ctx.WriteString("aggstar")
		ctx.WriteString(strconv.FormatBool(node.Aggstar))
	}

	if node.Aggtranstype != 0 {
		ctx.WriteString("aggtranstype")
		ctx.WriteString(strconv.Itoa(int(node.Aggtranstype)))
	}

	if node.Aggtype != 0 {
		ctx.WriteString("aggtype")
		ctx.WriteString(strconv.Itoa(int(node.Aggtype)))
	}

	if node.Aggvariadic {
		ctx.WriteString("aggvariadic")
		ctx.WriteString(strconv.FormatBool(node.Aggvariadic))
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

	if node.Inputcollid != 0 {
		ctx.WriteString("inputcollid")
		ctx.WriteString(strconv.Itoa(int(node.Inputcollid)))
	}

	// Intentionally ignoring node.Location for fingerprinting

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
