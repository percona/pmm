// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeTblFunction -
 *	  RangeTblEntry subsidiary data for one function in a FUNCTION RTE.
 *
 * If the function had a column definition list (required for an
 * otherwise-unspecified RECORD result), funccolnames lists the names given
 * in the definition list, funccoltypes lists their declared column types,
 * funccoltypmods lists their typmods, funccolcollations their collations.
 * Otherwise, those fields are NIL.
 *
 * Notice we don't attempt to store info about the results of functions
 * returning named composite types, because those can change from time to
 * time.  We do however remember how many columns we thought the type had
 * (including dropped columns!), so that we can successfully ignore any
 * columns added after the query was parsed.
 */
type RangeTblFunction struct {
	Funcexpr     Node `json:"funcexpr"`     /* expression tree for func call */
	Funccolcount int  `json:"funccolcount"` /* number of columns it contributes to RTE */

	/* These fields record the contents of a column definition list, if any: */
	Funccolnames      List `json:"funccolnames"`      /* column names (list of String) */
	Funccoltypes      List `json:"funccoltypes"`      /* OID list of column type OIDs */
	Funccoltypmods    List `json:"funccoltypmods"`    /* integer list of column typmods */
	Funccolcollations List `json:"funccolcollations"` /* OID list of column collation OIDs */

	/* This is set during planning for use by the executor: */
	Funcparams []uint32 `json:"funcparams"` /* PARAM_EXEC Param IDs affecting this func */
}

func (node RangeTblFunction) MarshalJSON() ([]byte, error) {
	type RangeTblFunctionMarshalAlias RangeTblFunction
	return json.Marshal(map[string]interface{}{
		"RangeTblFunction": (*RangeTblFunctionMarshalAlias)(&node),
	})
}

func (node *RangeTblFunction) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["funcexpr"] != nil {
		node.Funcexpr, err = UnmarshalNodeJSON(fields["funcexpr"])
		if err != nil {
			return
		}
	}

	if fields["funccolcount"] != nil {
		err = json.Unmarshal(fields["funccolcount"], &node.Funccolcount)
		if err != nil {
			return
		}
	}

	if fields["funccolnames"] != nil {
		node.Funccolnames.Items, err = UnmarshalNodeArrayJSON(fields["funccolnames"])
		if err != nil {
			return
		}
	}

	if fields["funccoltypes"] != nil {
		node.Funccoltypes.Items, err = UnmarshalNodeArrayJSON(fields["funccoltypes"])
		if err != nil {
			return
		}
	}

	if fields["funccoltypmods"] != nil {
		node.Funccoltypmods.Items, err = UnmarshalNodeArrayJSON(fields["funccoltypmods"])
		if err != nil {
			return
		}
	}

	if fields["funccolcollations"] != nil {
		node.Funccolcollations.Items, err = UnmarshalNodeArrayJSON(fields["funccolcollations"])
		if err != nil {
			return
		}
	}

	if fields["funcparams"] != nil {
		err = json.Unmarshal(fields["funcparams"], &node.Funcparams)
		if err != nil {
			return
		}
	}

	return
}
