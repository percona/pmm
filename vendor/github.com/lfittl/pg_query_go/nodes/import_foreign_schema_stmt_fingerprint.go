// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ImportForeignSchemaStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ImportForeignSchemaStmt")

	if int(node.ListType) != 0 {
		ctx.WriteString("list_type")
		ctx.WriteString(strconv.Itoa(int(node.ListType)))
	}

	if node.LocalSchema != nil {
		ctx.WriteString("local_schema")
		ctx.WriteString(*node.LocalSchema)
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

	if node.RemoteSchema != nil {
		ctx.WriteString("remote_schema")
		ctx.WriteString(*node.RemoteSchema)
	}

	if node.ServerName != nil {
		ctx.WriteString("server_name")
		ctx.WriteString(*node.ServerName)
	}

	if len(node.TableList.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TableList.Fingerprint(&subCtx, node, "TableList")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("table_list")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
