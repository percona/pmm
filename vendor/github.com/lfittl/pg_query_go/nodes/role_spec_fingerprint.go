// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RoleSpec) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RoleSpec")
	// Intentionally ignoring node.Location for fingerprinting

	if node.Rolename != nil {
		ctx.WriteString("rolename")
		ctx.WriteString(*node.Rolename)
	}

	if int(node.Roletype) != 0 {
		ctx.WriteString("roletype")
		ctx.WriteString(strconv.Itoa(int(node.Roletype)))
	}
}
