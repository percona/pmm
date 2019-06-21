// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TableFunc - node for a table function, such as XMLTABLE.
 */
type TableFunc struct {
	NsUris        List     `json:"ns_uris"`       /* list of namespace uri */
	NsNames       List     `json:"ns_names"`      /* list of namespace names */
	Docexpr       Node     `json:"docexpr"`       /* input document expression */
	Rowexpr       Node     `json:"rowexpr"`       /* row filter expression */
	Colnames      List     `json:"colnames"`      /* column names (list of String) */
	Coltypes      List     `json:"coltypes"`      /* OID list of column type OIDs */
	Coltypmods    List     `json:"coltypmods"`    /* integer list of column typmods */
	Colcollations List     `json:"colcollations"` /* OID list of column collation OIDs */
	Colexprs      List     `json:"colexprs"`      /* list of column filter expressions */
	Coldefexprs   List     `json:"coldefexprs"`   /* list of column default expressions */
	Notnulls      []uint32 `json:"notnulls"`      /* nullability flag for each output column */
	Ordinalitycol int      `json:"ordinalitycol"` /* counts from 0; -1 if none specified */
	Location      int      `json:"location"`      /* token location, or -1 if unknown */
}

func (node TableFunc) MarshalJSON() ([]byte, error) {
	type TableFuncMarshalAlias TableFunc
	return json.Marshal(map[string]interface{}{
		"TableFunc": (*TableFuncMarshalAlias)(&node),
	})
}

func (node *TableFunc) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["ns_uris"] != nil {
		node.NsUris.Items, err = UnmarshalNodeArrayJSON(fields["ns_uris"])
		if err != nil {
			return
		}
	}

	if fields["ns_names"] != nil {
		node.NsNames.Items, err = UnmarshalNodeArrayJSON(fields["ns_names"])
		if err != nil {
			return
		}
	}

	if fields["docexpr"] != nil {
		node.Docexpr, err = UnmarshalNodeJSON(fields["docexpr"])
		if err != nil {
			return
		}
	}

	if fields["rowexpr"] != nil {
		node.Rowexpr, err = UnmarshalNodeJSON(fields["rowexpr"])
		if err != nil {
			return
		}
	}

	if fields["colnames"] != nil {
		node.Colnames.Items, err = UnmarshalNodeArrayJSON(fields["colnames"])
		if err != nil {
			return
		}
	}

	if fields["coltypes"] != nil {
		node.Coltypes.Items, err = UnmarshalNodeArrayJSON(fields["coltypes"])
		if err != nil {
			return
		}
	}

	if fields["coltypmods"] != nil {
		node.Coltypmods.Items, err = UnmarshalNodeArrayJSON(fields["coltypmods"])
		if err != nil {
			return
		}
	}

	if fields["colcollations"] != nil {
		node.Colcollations.Items, err = UnmarshalNodeArrayJSON(fields["colcollations"])
		if err != nil {
			return
		}
	}

	if fields["colexprs"] != nil {
		node.Colexprs.Items, err = UnmarshalNodeArrayJSON(fields["colexprs"])
		if err != nil {
			return
		}
	}

	if fields["coldefexprs"] != nil {
		node.Coldefexprs.Items, err = UnmarshalNodeArrayJSON(fields["coldefexprs"])
		if err != nil {
			return
		}
	}

	if fields["notnulls"] != nil {
		err = json.Unmarshal(fields["notnulls"], &node.Notnulls)
		if err != nil {
			return
		}
	}

	if fields["ordinalitycol"] != nil {
		err = json.Unmarshal(fields["ordinalitycol"], &node.Ordinalitycol)
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
