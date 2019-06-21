// Auto-generated - DO NOT EDIT

package pg_query

import (
	"sort"
	"strconv"
)

func (node SelectStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("SelectStmt")

	if node.All {
		ctx.WriteString("all")
		ctx.WriteString(strconv.FormatBool(node.All))
	}

	if len(node.DistinctClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.DistinctClause.Fingerprint(&subCtx, node, "DistinctClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("distinctClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.FromClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.FromClause.Fingerprint(&subCtx, node, "FromClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("fromClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.GroupClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.GroupClause.Fingerprint(&subCtx, node, "GroupClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("groupClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.HavingClause != nil {
		subCtx := FingerprintSubContext{}
		node.HavingClause.Fingerprint(&subCtx, node, "HavingClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("havingClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IntoClause != nil {
		subCtx := FingerprintSubContext{}
		node.IntoClause.Fingerprint(&subCtx, node, "IntoClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("intoClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Larg != nil {
		subCtx := FingerprintSubContext{}
		node.Larg.Fingerprint(&subCtx, node, "Larg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("larg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.LimitCount != nil {
		subCtx := FingerprintSubContext{}
		node.LimitCount.Fingerprint(&subCtx, node, "LimitCount")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("limitCount")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.LimitOffset != nil {
		subCtx := FingerprintSubContext{}
		node.LimitOffset.Fingerprint(&subCtx, node, "LimitOffset")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("limitOffset")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.LockingClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.LockingClause.Fingerprint(&subCtx, node, "LockingClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("lockingClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Op) != 0 {
		ctx.WriteString("op")
		ctx.WriteString(strconv.Itoa(int(node.Op)))
	}

	if node.Rarg != nil {
		subCtx := FingerprintSubContext{}
		node.Rarg.Fingerprint(&subCtx, node, "Rarg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("rarg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.SortClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.SortClause.Fingerprint(&subCtx, node, "SortClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("sortClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.TargetList.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TargetList.Fingerprint(&subCtx, node, "TargetList")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("targetList")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.ValuesLists) > 0 {
		var itemsFingerprints FingerprintSubContextSlice

		for _, nodeList := range node.ValuesLists {
			subCtx := FingerprintSubContext{}
			for _, subNode := range nodeList {
				subNode.Fingerprint(&subCtx, node, "ValuesLists")
			}
			itemsFingerprints.AddIfUnique(subCtx)
		}

		sort.Sort(itemsFingerprints)

		if len(itemsFingerprints) > 0 {
			firstItem := true
			for _, fingerprint := range itemsFingerprints {
				for _, part := range fingerprint.parts {
					if firstItem {
						ctx.WriteString("valuesLists")
						firstItem = false
					}
					ctx.WriteString(part)
				}
			}
		}
	}

	if node.WhereClause != nil {
		subCtx := FingerprintSubContext{}
		node.WhereClause.Fingerprint(&subCtx, node, "WhereClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("whereClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.WindowClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.WindowClause.Fingerprint(&subCtx, node, "WindowClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("windowClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.WithClause != nil {
		subCtx := FingerprintSubContext{}
		node.WithClause.Fingerprint(&subCtx, node, "WithClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("withClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
