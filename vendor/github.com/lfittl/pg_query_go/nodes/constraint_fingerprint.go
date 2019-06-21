// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node Constraint) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("Constraint")

	if node.AccessMethod != nil {
		ctx.WriteString("access_method")
		ctx.WriteString(*node.AccessMethod)
	}

	if node.Conname != nil {
		ctx.WriteString("conname")
		ctx.WriteString(*node.Conname)
	}

	if int(node.Contype) != 0 {
		ctx.WriteString("contype")
		ctx.WriteString(strconv.Itoa(int(node.Contype)))
	}

	if node.CookedExpr != nil {
		ctx.WriteString("cooked_expr")
		ctx.WriteString(*node.CookedExpr)
	}

	if node.Deferrable {
		ctx.WriteString("deferrable")
		ctx.WriteString(strconv.FormatBool(node.Deferrable))
	}

	if len(node.Exclusions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Exclusions.Fingerprint(&subCtx, node, "Exclusions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("exclusions")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.FkAttrs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.FkAttrs.Fingerprint(&subCtx, node, "FkAttrs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("fk_attrs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.FkDelAction != 0 {
		ctx.WriteString("fk_del_action")
		ctx.WriteString(string(node.FkDelAction))

	}

	if node.FkMatchtype != 0 {
		ctx.WriteString("fk_matchtype")
		ctx.WriteString(string(node.FkMatchtype))

	}

	if node.FkUpdAction != 0 {
		ctx.WriteString("fk_upd_action")
		ctx.WriteString(string(node.FkUpdAction))

	}

	if node.GeneratedWhen != 0 {
		ctx.WriteString("generated_when")
		ctx.WriteString(string(node.GeneratedWhen))

	}

	if node.Indexname != nil {
		ctx.WriteString("indexname")
		ctx.WriteString(*node.Indexname)
	}

	if node.Indexspace != nil {
		ctx.WriteString("indexspace")
		ctx.WriteString(*node.Indexspace)
	}

	if node.Initdeferred {
		ctx.WriteString("initdeferred")
		ctx.WriteString(strconv.FormatBool(node.Initdeferred))
	}

	if node.InitiallyValid {
		ctx.WriteString("initially_valid")
		ctx.WriteString(strconv.FormatBool(node.InitiallyValid))
	}

	if node.IsNoInherit {
		ctx.WriteString("is_no_inherit")
		ctx.WriteString(strconv.FormatBool(node.IsNoInherit))
	}

	if len(node.Keys.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Keys.Fingerprint(&subCtx, node, "Keys")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("keys")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.OldConpfeqop.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.OldConpfeqop.Fingerprint(&subCtx, node, "OldConpfeqop")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("old_conpfeqop")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.OldPktableOid != 0 {
		ctx.WriteString("old_pktable_oid")
		ctx.WriteString(strconv.Itoa(int(node.OldPktableOid)))
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

	if len(node.PkAttrs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.PkAttrs.Fingerprint(&subCtx, node, "PkAttrs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("pk_attrs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Pktable != nil {
		subCtx := FingerprintSubContext{}
		node.Pktable.Fingerprint(&subCtx, node, "Pktable")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("pktable")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.RawExpr != nil {
		subCtx := FingerprintSubContext{}
		node.RawExpr.Fingerprint(&subCtx, node, "RawExpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("raw_expr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.SkipValidation {
		ctx.WriteString("skip_validation")
		ctx.WriteString(strconv.FormatBool(node.SkipValidation))
	}

	if node.WhereClause != nil {
		subCtx := FingerprintSubContext{}
		node.WhereClause.Fingerprint(&subCtx, node, "WhereClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("where_clause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
