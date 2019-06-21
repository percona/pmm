// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node GrantRoleStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("GrantRoleStmt")

	if node.AdminOpt {
		ctx.WriteString("admin_opt")
		ctx.WriteString(strconv.FormatBool(node.AdminOpt))
	}

	if int(node.Behavior) != 0 {
		ctx.WriteString("behavior")
		ctx.WriteString(strconv.Itoa(int(node.Behavior)))
	}

	if len(node.GrantedRoles.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.GrantedRoles.Fingerprint(&subCtx, node, "GrantedRoles")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("granted_roles")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.GranteeRoles.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.GranteeRoles.Fingerprint(&subCtx, node, "GranteeRoles")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("grantee_roles")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Grantor != nil {
		subCtx := FingerprintSubContext{}
		node.Grantor.Fingerprint(&subCtx, node, "Grantor")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("grantor")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IsGrant {
		ctx.WriteString("is_grant")
		ctx.WriteString(strconv.FormatBool(node.IsGrant))
	}
}
