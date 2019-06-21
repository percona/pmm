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
type ParamListInfoData struct {
	ParamFetchArg  interface{} `json:"paramFetchArg"`
	ParserSetupArg interface{} `json:"parserSetupArg"`
	NumParams      int         `json:"numParams"` /* number of ParamExternDatas following */
	ParamMask      []uint32    `json:"paramMask"` /* if non-NULL, can ignore omitted params */
}

func (node ParamListInfoData) MarshalJSON() ([]byte, error) {
	type ParamListInfoDataMarshalAlias ParamListInfoData
	return json.Marshal(map[string]interface{}{
		"ParamListInfoData": (*ParamListInfoDataMarshalAlias)(&node),
	})
}

func (node *ParamListInfoData) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["paramFetchArg"] != nil {
		err = json.Unmarshal(fields["paramFetchArg"], &node.ParamFetchArg)
		if err != nil {
			return
		}
	}

	if fields["parserSetupArg"] != nil {
		err = json.Unmarshal(fields["parserSetupArg"], &node.ParserSetupArg)
		if err != nil {
			return
		}
	}

	if fields["numParams"] != nil {
		err = json.Unmarshal(fields["numParams"], &node.NumParams)
		if err != nil {
			return
		}
	}

	if fields["paramMask"] != nil {
		err = json.Unmarshal(fields["paramMask"], &node.ParamMask)
		if err != nil {
			return
		}
	}

	return
}
