// Auto-generated - DO NOT EDIT

package pg_query

import "sort"

func (node List) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	if parentFieldName == "FromClause" || parentFieldName == "TargetList" || parentFieldName == "Cols" || parentFieldName == "Rexpr" {
		var itemsFingerprints FingerprintSubContextSlice

		for _, subNode := range node.Items {
			if subNode != nil {
				subCtx := FingerprintSubContext{}
				subNode.Fingerprint(&subCtx, parentNode, parentFieldName)
				itemsFingerprints.AddIfUnique(subCtx)
			}
		}

		sort.Sort(itemsFingerprints)

		for _, fingerprint := range itemsFingerprints {
			for _, part := range fingerprint.parts {
				ctx.WriteString(part)
			}
		}
	} else {
		for _, subNode := range node.Items {
			if subNode != nil {
				subNode.Fingerprint(ctx, parentNode, parentFieldName)
			}
		}
	}
}
