// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * FuncCall - a function or aggregate invocation
 *
 * agg_order (if not NIL) indicates we saw 'foo(... ORDER BY ...)', or if
 * agg_within_group is true, it was 'foo(...) WITHIN GROUP (ORDER BY ...)'.
 * agg_star indicates we saw a 'foo(*)' construct, while agg_distinct
 * indicates we saw 'foo(DISTINCT ...)'.  In any of these cases, the
 * construct *must* be an aggregate call.  Otherwise, it might be either an
 * aggregate or some other kind of function.  However, if FILTER or OVER is
 * present it had better be an aggregate or window function.
 *
 * Normally, you'd initialize this via makeFuncCall() and then only change the
 * parts of the struct its defaults don't match afterwards, as needed.
 */
type FuncCall struct {
	Funcname       List       `json:"funcname"`         /* qualified name of function */
	Args           List       `json:"args"`             /* the arguments (list of exprs) */
	AggOrder       List       `json:"agg_order"`        /* ORDER BY (list of SortBy) */
	AggFilter      Node       `json:"agg_filter"`       /* FILTER clause, if any */
	AggWithinGroup bool       `json:"agg_within_group"` /* ORDER BY appeared in WITHIN GROUP */
	AggStar        bool       `json:"agg_star"`         /* argument was really '*' */
	AggDistinct    bool       `json:"agg_distinct"`     /* arguments were labeled DISTINCT */
	FuncVariadic   bool       `json:"func_variadic"`    /* last argument was labeled VARIADIC */
	Over           *WindowDef `json:"over"`             /* OVER clause, if any */
	Location       int        `json:"location"`         /* token location, or -1 if unknown */
}

func (node FuncCall) MarshalJSON() ([]byte, error) {
	type FuncCallMarshalAlias FuncCall
	return json.Marshal(map[string]interface{}{
		"FuncCall": (*FuncCallMarshalAlias)(&node),
	})
}

func (node *FuncCall) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["funcname"] != nil {
		node.Funcname.Items, err = UnmarshalNodeArrayJSON(fields["funcname"])
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

	if fields["agg_order"] != nil {
		node.AggOrder.Items, err = UnmarshalNodeArrayJSON(fields["agg_order"])
		if err != nil {
			return
		}
	}

	if fields["agg_filter"] != nil {
		node.AggFilter, err = UnmarshalNodeJSON(fields["agg_filter"])
		if err != nil {
			return
		}
	}

	if fields["agg_within_group"] != nil {
		err = json.Unmarshal(fields["agg_within_group"], &node.AggWithinGroup)
		if err != nil {
			return
		}
	}

	if fields["agg_star"] != nil {
		err = json.Unmarshal(fields["agg_star"], &node.AggStar)
		if err != nil {
			return
		}
	}

	if fields["agg_distinct"] != nil {
		err = json.Unmarshal(fields["agg_distinct"], &node.AggDistinct)
		if err != nil {
			return
		}
	}

	if fields["func_variadic"] != nil {
		err = json.Unmarshal(fields["func_variadic"], &node.FuncVariadic)
		if err != nil {
			return
		}
	}

	if fields["over"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["over"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(WindowDef)
			node.Over = &val
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
