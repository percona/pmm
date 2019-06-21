// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TypeName) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TypeName")

	if len(node.ArrayBounds.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ArrayBounds.Fingerprint(&subCtx, node, "ArrayBounds")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arrayBounds")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.Names.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Names.Fingerprint(&subCtx, node, "Names")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("names")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.PctType {
		ctx.WriteString("pct_type")
		ctx.WriteString(strconv.FormatBool(node.PctType))
	}

	if node.Setof {
		ctx.WriteString("setof")
		ctx.WriteString(strconv.FormatBool(node.Setof))
	}

	if node.TypeOid != 0 {
		ctx.WriteString("typeOid")
		ctx.WriteString(strconv.Itoa(int(node.TypeOid)))
	}

	if node.Typemod != 0 {
		ctx.WriteString("typemod")
		ctx.WriteString(strconv.Itoa(int(node.Typemod)))
	}

	if len(node.Typmods.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Typmods.Fingerprint(&subCtx, node, "Typmods")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("typmods")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
