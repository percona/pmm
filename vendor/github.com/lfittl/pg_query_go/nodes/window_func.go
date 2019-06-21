// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * WindowFunc
 */
type WindowFunc struct {
	Xpr         Node  `json:"xpr"`
	Winfnoid    Oid   `json:"winfnoid"`    /* pg_proc Oid of the function */
	Wintype     Oid   `json:"wintype"`     /* type Oid of result of the window function */
	Wincollid   Oid   `json:"wincollid"`   /* OID of collation of result */
	Inputcollid Oid   `json:"inputcollid"` /* OID of collation that function should use */
	Args        List  `json:"args"`        /* arguments to the window function */
	Aggfilter   Node  `json:"aggfilter"`   /* FILTER expression, if any */
	Winref      Index `json:"winref"`      /* index of associated WindowClause */
	Winstar     bool  `json:"winstar"`     /* TRUE if argument list was really '*' */
	Winagg      bool  `json:"winagg"`      /* is function a simple aggregate? */
	Location    int   `json:"location"`    /* token location, or -1 if unknown */
}

func (node WindowFunc) MarshalJSON() ([]byte, error) {
	type WindowFuncMarshalAlias WindowFunc
	return json.Marshal(map[string]interface{}{
		"WindowFunc": (*WindowFuncMarshalAlias)(&node),
	})
}

func (node *WindowFunc) UnmarshalJSON(input []byte) (err error) {
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

	if fields["winfnoid"] != nil {
		err = json.Unmarshal(fields["winfnoid"], &node.Winfnoid)
		if err != nil {
			return
		}
	}

	if fields["wintype"] != nil {
		err = json.Unmarshal(fields["wintype"], &node.Wintype)
		if err != nil {
			return
		}
	}

	if fields["wincollid"] != nil {
		err = json.Unmarshal(fields["wincollid"], &node.Wincollid)
		if err != nil {
			return
		}
	}

	if fields["inputcollid"] != nil {
		err = json.Unmarshal(fields["inputcollid"], &node.Inputcollid)
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["aggfilter"] != nil {
		node.Aggfilter, err = UnmarshalNodeJSON(fields["aggfilter"])
		if err != nil {
			return
		}
	}

	if fields["winref"] != nil {
		err = json.Unmarshal(fields["winref"], &node.Winref)
		if err != nil {
			return
		}
	}

	if fields["winstar"] != nil {
		err = json.Unmarshal(fields["winstar"], &node.Winstar)
		if err != nil {
			return
		}
	}

	if fields["winagg"] != nil {
		err = json.Unmarshal(fields["winagg"], &node.Winagg)
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
