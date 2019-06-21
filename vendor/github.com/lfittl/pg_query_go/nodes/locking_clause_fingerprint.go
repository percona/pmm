// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node LockingClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("LockingClause")

	if len(node.LockedRels.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.LockedRels.Fingerprint(&subCtx, node, "LockedRels")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("lockedRels")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Strength) != 0 {
		ctx.WriteString("strength")
		ctx.WriteString(strconv.Itoa(int(node.Strength)))
	}

	if int(node.WaitPolicy) != 0 {
		ctx.WriteString("waitPolicy")
		ctx.WriteString(strconv.Itoa(int(node.WaitPolicy)))
	}
}
