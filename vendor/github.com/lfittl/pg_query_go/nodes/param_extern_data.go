// Auto-generated from postgres/src/include/nodes/params.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 *	  ParamListInfo
 *
 *	  ParamListInfo arrays are used to pass parameters into the executor
 *	  for parameterized plans.  Each entry in the array defines the value
 *	  to be substituted for a PARAM_EXTERN parameter.  The "paramid"
 *	  of a PARAM_EXTERN Param can range from 1 to numParams.
 *
 *	  Although parameter numbers are normally consecutive, we allow
 *	  ptype == InvalidOid to signal an unused array entry.
 *
 *	  pflags is a flags field.  Currently the only used bit is:
 *	  PARAM_FLAG_CONST signals the planner that it may treat this parameter
 *	  as a constant (i.e., generate a plan that works only for this value
 *	  of the parameter).
 *
 *	  There are two hook functions that can be associated with a ParamListInfo
 *	  array to support dynamic parameter handling.  First, if paramFetch
 *	  isn't null and the executor requires a value for an invalid parameter
 *	  (one with ptype == InvalidOid), the paramFetch hook is called to give
 *	  it a chance to fill in the parameter value.  Second, a parserSetup
 *	  hook can be supplied to re-instantiate the original parsing hooks if
 *	  a query needs to be re-parsed/planned (as a substitute for supposing
 *	  that the current ptype values represent a fixed set of parameter types).

 *	  Although the data structure is really an array, not a list, we keep
 *	  the old typedef name to avoid unnecessary code changes.
 * ----------------
 */
type ParamExternData struct {
	Value  Datum  `json:"value"`  /* parameter value */
	Isnull bool   `json:"isnull"` /* is it NULL? */
	Pflags uint16 `json:"pflags"` /* flag bits, see above */
	Ptype  Oid    `json:"ptype"`  /* parameter's datatype, or 0 */
}

func (node ParamExternData) MarshalJSON() ([]byte, error) {
	type ParamExternDataMarshalAlias ParamExternData
	return json.Marshal(map[string]interface{}{
		"ParamExternData": (*ParamExternDataMarshalAlias)(&node),
	})
}

func (node *ParamExternData) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["value"] != nil {
		err = json.Unmarshal(fields["value"], &node.Value)
		if err != nil {
			return
		}
	}

	if fields["isnull"] != nil {
		err = json.Unmarshal(fields["isnull"], &node.Isnull)
		if err != nil {
			return
		}
	}

	if fields["pflags"] != nil {
		err = json.Unmarshal(fields["pflags"], &node.Pflags)
		if err != nil {
			return
		}
	}

	if fields["ptype"] != nil {
		err = json.Unmarshal(fields["ptype"], &node.Ptype)
		if err != nil {
			return
		}
	}

	return
}
