// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * FuncExpr - expression node for a function call
 */
type FuncExpr struct {
	Xpr            Node `json:"xpr"`
	Funcid         Oid  `json:"funcid"`         /* PG_PROC OID of the function */
	Funcresulttype Oid  `json:"funcresulttype"` /* PG_TYPE OID of result value */
	Funcretset     bool `json:"funcretset"`     /* true if function returns set */
	Funcvariadic   bool `json:"funcvariadic"`   /* true if variadic arguments have been
	 * combined into an array last argument */

	Funcformat  CoercionForm `json:"funcformat"`  /* how to display this function call */
	Funccollid  Oid          `json:"funccollid"`  /* OID of collation of result */
	Inputcollid Oid          `json:"inputcollid"` /* OID of collation that function should use */
	Args        List         `json:"args"`        /* arguments to the function */
	Location    int          `json:"location"`    /* token location, or -1 if unknown */
}

func (node FuncExpr) MarshalJSON() ([]byte, error) {
	type FuncExprMarshalAlias FuncExpr
	return json.Marshal(map[string]interface{}{
		"FuncExpr": (*FuncExprMarshalAlias)(&node),
	})
}

func (node *FuncExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["funcid"] != nil {
		err = json.Unmarshal(fields["funcid"], &node.Funcid)
		if err != nil {
			return
		}
	}

	if fields["funcresulttype"] != nil {
		err = json.Unmarshal(fields["funcresulttype"], &node.Funcresulttype)
		if err != nil {
			return
		}
	}

	if fields["funcretset"] != nil {
		err = json.Unmarshal(fields["funcretset"], &node.Funcretset)
		if err != nil {
			return
		}
	}

	if fields["funcvariadic"] != nil {
		err = json.Unmarshal(fields["funcvariadic"], &node.Funcvariadic)
		if err != nil {
			return
		}
	}

	if fields["funcformat"] != nil {
		err = json.Unmarshal(fields["funcformat"], &node.Funcformat)
		if err != nil {
			return
		}
	}

	if fields["funccollid"] != nil {
		err = json.Unmarshal(fields["funccollid"], &node.Funccollid)
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

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
		if err != nil {
			return
		}
	}

	return
}
