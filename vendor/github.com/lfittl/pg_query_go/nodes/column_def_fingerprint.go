// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ColumnDef) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ColumnDef")

	if node.CollClause != nil {
		subCtx := FingerprintSubContext{}
		node.CollClause.Fingerprint(&subCtx, node, "CollClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("collClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.CollOid != 0 {
		ctx.WriteString("collOid")
		ctx.WriteString(strconv.Itoa(int(node.CollOid)))
	}

	if node.Colname != nil {
		ctx.WriteString("colname")
		ctx.WriteString(*node.Colname)
	}

	if len(node.Constraints.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Constraints.Fingerprint(&subCtx, node, "Constraints")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("constraints")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.CookedDefault != nil {
		subCtx := FingerprintSubContext{}
		node.CookedDefault.Fingerprint(&subCtx, node, "CookedDefault")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("cooked_default")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Fdwoptions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Fdwoptions.Fingerprint(&subCtx, node, "Fdwoptions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("fdwoptions")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Identity != 0 {
		ctx.WriteString("identity")
		ctx.WriteString(string(node.Identity))

	}

	if node.Inhcount != 0 {
		ctx.WriteString("inhcount")
		ctx.WriteString(strconv.Itoa(int(node.Inhcount)))
	}

	if node.IsFromParent {
		ctx.WriteString("is_from_parent")
		ctx.WriteString(strconv.FormatBool(node.IsFromParent))
	}

	if node.IsFromType {
		ctx.WriteString("is_from_type")
		ctx.WriteString(strconv.FormatBool(node.IsFromType))
	}

	if node.IsLocal {
		ctx.WriteString("is_local")
		ctx.WriteString(strconv.FormatBool(node.IsLocal))
	}

	if node.IsNotNull {
		ctx.WriteString("is_not_null")
		ctx.WriteString(strconv.FormatBool(node.IsNotNull))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if node.RawDefault != nil {
		subCtx := FingerprintSubContext{}
		node.RawDefault.Fingerprint(&subCtx, node, "RawDefault")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("raw_default")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Storage != 0 {
		ctx.WriteString("storage")
		ctx.WriteString(string(node.Storage))

	}

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
