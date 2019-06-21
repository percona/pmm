// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeTblEntry) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeTblEntry")

	if node.Alias != nil {
		subCtx := FingerprintSubContext{}
		node.Alias.Fingerprint(&subCtx, node, "Alias")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("alias")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.CheckAsUser != 0 {
		ctx.WriteString("checkAsUser")
		ctx.WriteString(strconv.Itoa(int(node.CheckAsUser)))
	}

	if len(node.Colcollations.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Colcollations.Fingerprint(&subCtx, node, "Colcollations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colcollations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Coltypes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Coltypes.Fingerprint(&subCtx, node, "Coltypes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coltypes")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Coltypmods.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Coltypmods.Fingerprint(&subCtx, node, "Coltypmods")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coltypmods")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Ctelevelsup != 0 {
		ctx.WriteString("ctelevelsup")
		ctx.WriteString(strconv.Itoa(int(node.Ctelevelsup)))
	}

	if node.Ctename != nil {
		ctx.WriteString("ctename")
		ctx.WriteString(*node.Ctename)
	}

	if node.Enrname != nil {
		ctx.WriteString("enrname")
		ctx.WriteString(*node.Enrname)
	}

	ctx.WriteString("enrtuples")
	ctx.WriteString(strconv.FormatFloat(float64(node.Enrtuples), 'E', -1, 64))

	if node.Eref != nil {
		subCtx := FingerprintSubContext{}
		node.Eref.Fingerprint(&subCtx, node, "Eref")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("eref")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Funcordinality {
		ctx.WriteString("funcordinality")
		ctx.WriteString(strconv.FormatBool(node.Funcordinality))
	}

	if len(node.Functions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Functions.Fingerprint(&subCtx, node, "Functions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("functions")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.InFromCl {
		ctx.WriteString("inFromCl")
		ctx.WriteString(strconv.FormatBool(node.InFromCl))
	}

	if node.Inh {
		ctx.WriteString("inh")
		ctx.WriteString(strconv.FormatBool(node.Inh))
	}

	ctx.WriteString("insertedCols")
	for _, val := range node.InsertedCols {
		ctx.WriteString(strconv.Itoa(int(val)))
	}

	if len(node.Joinaliasvars.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Joinaliasvars.Fingerprint(&subCtx, node, "Joinaliasvars")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("joinaliasvars")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Jointype) != 0 {
		ctx.WriteString("jointype")
		ctx.WriteString(strconv.Itoa(int(node.Jointype)))
	}

	if node.Lateral {
		ctx.WriteString("lateral")
		ctx.WriteString(strconv.FormatBool(node.Lateral))
	}

	if node.Relid != 0 {
		ctx.WriteString("relid")
		ctx.WriteString(strconv.Itoa(int(node.Relid)))
	}

	if node.Relkind != 0 {
		ctx.WriteString("relkind")
		ctx.WriteString(string(node.Relkind))

	}

	if node.RequiredPerms != 0 {
		ctx.WriteString("requiredPerms")
		ctx.WriteString(strconv.Itoa(int(node.RequiredPerms)))
	}

	if int(node.Rtekind) != 0 {
		ctx.WriteString("rtekind")
		ctx.WriteString(strconv.Itoa(int(node.Rtekind)))
	}

	if len(node.SecurityQuals.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.SecurityQuals.Fingerprint(&subCtx, node, "SecurityQuals")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("securityQuals")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.SecurityBarrier {
		ctx.WriteString("security_barrier")
		ctx.WriteString(strconv.FormatBool(node.SecurityBarrier))
	}

	ctx.WriteString("selectedCols")
	for _, val := range node.SelectedCols {
		ctx.WriteString(strconv.Itoa(int(val)))
	}

	if node.SelfReference {
		ctx.WriteString("self_reference")
		ctx.WriteString(strconv.FormatBool(node.SelfReference))
	}

	if node.Subquery != nil {
		subCtx := FingerprintSubContext{}
		node.Subquery.Fingerprint(&subCtx, node, "Subquery")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("subquery")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Tablefunc != nil {
		subCtx := FingerprintSubContext{}
		node.Tablefunc.Fingerprint(&subCtx, node, "Tablefunc")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("tablefunc")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Tablesample != nil {
		subCtx := FingerprintSubContext{}
		node.Tablesample.Fingerprint(&subCtx, node, "Tablesample")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("tablesample")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	ctx.WriteString("updatedCols")
	for _, val := range node.UpdatedCols {
		ctx.WriteString(strconv.Itoa(int(val)))
	}

	if len(node.ValuesLists.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ValuesLists.Fingerprint(&subCtx, node, "ValuesLists")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("values_lists")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
