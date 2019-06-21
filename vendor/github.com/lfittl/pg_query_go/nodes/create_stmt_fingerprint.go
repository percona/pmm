// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateStmt")

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

	if node.IfNotExists {
		ctx.WriteString("if_not_exists")
		ctx.WriteString(strconv.FormatBool(node.IfNotExists))
	}

	if len(node.InhRelations.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.InhRelations.Fingerprint(&subCtx, node, "InhRelations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("inhRelations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.OfTypename != nil {
		subCtx := FingerprintSubContext{}
		node.OfTypename.Fingerprint(&subCtx, node, "OfTypename")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ofTypename")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Oncommit) != 0 {
		ctx.WriteString("oncommit")
		ctx.WriteString(strconv.Itoa(int(node.Oncommit)))
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

	if node.Partbound != nil {
		subCtx := FingerprintSubContext{}
		node.Partbound.Fingerprint(&subCtx, node, "Partbound")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("partbound")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Partspec != nil {
		subCtx := FingerprintSubContext{}
		node.Partspec.Fingerprint(&subCtx, node, "Partspec")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("partspec")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Relation != nil {
		subCtx := FingerprintSubContext{}
		node.Relation.Fingerprint(&subCtx, node, "Relation")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("relation")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.TableElts.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TableElts.Fingerprint(&subCtx, node, "TableElts")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("tableElts")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Tablespacename != nil {
		ctx.WriteString("tablespacename")
		ctx.WriteString(*node.Tablespacename)
	}
}
