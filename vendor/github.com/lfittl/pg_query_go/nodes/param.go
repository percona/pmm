// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Param
 *
 *		paramkind specifies the kind of parameter. The possible values
 *		for this field are:
 *
 *		PARAM_EXTERN:  The parameter value is supplied from outside the plan.
 *				Such parameters are numbered from 1 to n.
 *
 *		PARAM_EXEC:  The parameter is an internal executor parameter, used
 *				for passing values into and out of sub-queries or from
 *				nestloop joins to their inner scans.
 *				For historical reasons, such parameters are numbered from 0.
 *				These numbers are independent of PARAM_EXTERN numbers.
 *
 *		PARAM_SUBLINK:	The parameter represents an output column of a SubLink
 *				node's sub-select.  The column number is contained in the
 *				`paramid' field.  (This type of Param is converted to
 *				PARAM_EXEC during planning.)
 *
 *		PARAM_MULTIEXPR:  Like PARAM_SUBLINK, the parameter represents an
 *				output column of a SubLink node's sub-select, but here, the
 *				SubLink is always a MULTIEXPR SubLink.  The high-order 16 bits
 *				of the `paramid' field contain the SubLink's subLinkId, and
 *				the low-order 16 bits contain the column number.  (This type
 *				of Param is also converted to PARAM_EXEC during planning.)
 */
type Param struct {
	Xpr         Node      `json:"xpr"`
	Paramkind   ParamKind `json:"paramkind"`   /* kind of parameter. See above */
	Paramid     int       `json:"paramid"`     /* numeric ID for parameter */
	Paramtype   Oid       `json:"paramtype"`   /* pg_type OID of parameter's datatype */
	Paramtypmod int32     `json:"paramtypmod"` /* typmod value, if known */
	Paramcollid Oid       `json:"paramcollid"` /* OID of collation, or InvalidOid if none */
	Location    int       `json:"location"`    /* token location, or -1 if unknown */
}

func (node Param) MarshalJSON() ([]byte, error) {
	type ParamMarshalAlias Param
	return json.Marshal(map[string]interface{}{
		"Param": (*ParamMarshalAlias)(&node),
	})
}

func (node *Param) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xpr"] != nil {
		node.Xpr, err = UnmarshalNodeJSON(fields["xpr"])
		if err != nil {
			return
		}
	}

	if fields["paramkind"] != nil {
		err = json.Unmarshal(fields["paramkind"], &node.Paramkind)
		if err != nil {
			return
		}
	}

	if fields["paramid"] != nil {
		err = json.Unmarshal(fields["paramid"], &node.Paramid)
		if err != nil {
			return
		}
	}

	if fields["paramtype"] != nil {
		err = json.Unmarshal(fields["paramtype"], &node.Paramtype)
		if err != nil {
			return
		}
	}

	if fields["paramtypmod"] != nil {
		err = json.Unmarshal(fields["paramtypmod"], &node.Paramtypmod)
		if err != nil {
			return
		}
	}

	if fields["paramcollid"] != nil {
		err = json.Unmarshal(fields["paramcollid"], &node.Paramcollid)
		if err != nil {
			return
		}
	}

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
		if err != nil {
			return
		}
	}

	return
}
