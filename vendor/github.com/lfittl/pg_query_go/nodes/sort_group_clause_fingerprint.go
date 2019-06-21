// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node SortGroupClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("SortGroupClause")

	if node.Eqop != 0 {
		ctx.WriteString("eqop")
		ctx.WriteString(strconv.Itoa(int(node.Eqop)))
	}

	if node.Hashable {
		ctx.WriteString("hashable")
		ctx.WriteString(strconv.FormatBool(node.Hashable))
	}

	if node.NullsFirst {
		ctx.WriteString("nulls_first")
		ctx.WriteString(strconv.FormatBool(node.NullsFirst))
	}

	if node.Sortop != 0 {
		ctx.WriteString("sortop")
		ctx.WriteString(strconv.Itoa(int(node.Sortop)))
	}

	if node.TleSortGroupRef != 0 {
		ctx.WriteString("tleSortGroupRef")
		ctx.WriteString(strconv.Itoa(int(node.TleSortGroupRef)))
	}
}
