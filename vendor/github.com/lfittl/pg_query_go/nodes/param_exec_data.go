// Auto-generated from postgres/src/include/nodes/params.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 *	  ParamExecData
 *
 *	  ParamExecData entries are used for executor internal parameters
 *	  (that is, values being passed into or out of a sub-query).  The
 *	  paramid of a PARAM_EXEC Param is a (zero-based) index into an
 *	  array of ParamExecData records, which is referenced through
 *	  es_param_exec_vals or ecxt_param_exec_vals.
 *
 *	  If execPlan is not NULL, it points to a SubPlanState node that needs
 *	  to be executed to produce the value.  (This is done so that we can have
 *	  lazy evaluation of InitPlans: they aren't executed until/unless a
 *	  result value is needed.)	Otherwise the value is assumed to be valid
 *	  when needed.
 * ----------------
 */
type ParamExecData struct {
	ExecPlan interface{} `json:"execPlan"` /* should be "SubPlanState *" */
	Value    Datum       `json:"value"`
	Isnull   bool        `json:"isnull"`
}

func (node ParamExecData) MarshalJSON() ([]byte, error) {
	type ParamExecDataMarshalAlias ParamExecData
	return json.Marshal(map[string]interface{}{
		"ParamExecData": (*ParamExecDataMarshalAlias)(&node),
	})
}

func (node *ParamExecData) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["execPlan"] != nil {
		err = json.Unmarshal(fields["execPlan"], &node.ExecPlan)
		if err != nil {
			return
		}
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

	return
}
