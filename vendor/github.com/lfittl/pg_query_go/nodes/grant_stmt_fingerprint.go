// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node GrantStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("GrantStmt")

	if int(node.Behavior) != 0 {
		ctx.WriteString("behavior")
		ctx.WriteString(strconv.Itoa(int(node.Behavior)))
	}

	if node.GrantOption {
		ctx.WriteString("grant_option")
		ctx.WriteString(strconv.FormatBool(node.GrantOption))
	}

	if len(node.Grantees.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Grantees.Fingerprint(&subCtx, node, "Grantees")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("grantees")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IsGrant {
		ctx.WriteString("is_grant")
		ctx.WriteString(strconv.FormatBool(node.IsGrant))
	}

	if len(node.Objects.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Objects.Fingerprint(&subCtx, node, "Objects")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("objects")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Objtype) != 0 {
		ctx.WriteString("objtype")
		ctx.WriteString(strconv.Itoa(int(node.Objtype)))
	}

	if len(node.Privileges.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Privileges.Fingerprint(&subCtx, node, "Privileges")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("privileges")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Targtype) != 0 {
		ctx.WriteString("targtype")
		ctx.WriteString(strconv.Itoa(int(node.Targtype)))
	}
}
