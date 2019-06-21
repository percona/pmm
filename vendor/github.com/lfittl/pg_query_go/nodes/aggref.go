// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Aggref
 *
 * The aggregate's args list is a targetlist, ie, a list of TargetEntry nodes.
 *
 * For a normal (non-ordered-set) aggregate, the non-resjunk TargetEntries
 * represent the aggregate's regular arguments (if any) and resjunk TLEs can
 * be added at the end to represent ORDER BY expressions that are not also
 * arguments.  As in a top-level Query, the TLEs can be marked with
 * ressortgroupref indexes to let them be referenced by SortGroupClause
 * entries in the aggorder and/or aggdistinct lists.  This represents ORDER BY
 * and DISTINCT operations to be applied to the aggregate input rows before
 * they are passed to the transition function.  The grammar only allows a
 * simple "DISTINCT" specifier for the arguments, but we use the full
 * query-level representation to allow more code sharing.
 *
 * For an ordered-set aggregate, the args list represents the WITHIN GROUP
 * (aggregated) arguments, all of which will be listed in the aggorder list.
 * DISTINCT is not supported in this case, so aggdistinct will be NIL.
 * The direct arguments appear in aggdirectargs (as a list of plain
 * expressions, not TargetEntry nodes).
 *
 * aggtranstype is the data type of the state transition values for this
 * aggregate (resolved to an actual type, if agg's transtype is polymorphic).
 * This is determined during planning and is InvalidOid before that.
 *
 * aggargtypes is an OID list of the data types of the direct and regular
 * arguments.  Normally it's redundant with the aggdirectargs and args lists,
 * but in a combining aggregate, it's not because the args list has been
 * replaced with a single argument representing the partial-aggregate
 * transition values.
 *
 * aggsplit indicates the expected partial-aggregation mode for the Aggref's
 * parent plan node.  It's always set to AGGSPLIT_SIMPLE in the parser, but
 * the planner might change it to something else.  We use this mainly as
 * a crosscheck that the Aggrefs match the plan; but note that when aggsplit
 * indicates a non-final mode, aggtype reflects the transition data type
 * not the SQL-level output type of the aggregate.
 */
type Aggref struct {
	Xpr           Node `json:"xpr"`
	Aggfnoid      Oid  `json:"aggfnoid"`      /* pg_proc Oid of the aggregate */
	Aggtype       Oid  `json:"aggtype"`       /* type Oid of result of the aggregate */
	Aggcollid     Oid  `json:"aggcollid"`     /* OID of collation of result */
	Inputcollid   Oid  `json:"inputcollid"`   /* OID of collation that function should use */
	Aggtranstype  Oid  `json:"aggtranstype"`  /* type Oid of aggregate's transition value */
	Aggargtypes   List `json:"aggargtypes"`   /* type Oids of direct and aggregated args */
	Aggdirectargs List `json:"aggdirectargs"` /* direct arguments, if an ordered-set agg */
	Args          List `json:"args"`          /* aggregated arguments and sort expressions */
	Aggorder      List `json:"aggorder"`      /* ORDER BY (list of SortGroupClause) */
	Aggdistinct   List `json:"aggdistinct"`   /* DISTINCT (list of SortGroupClause) */
	Aggfilter     Node `json:"aggfilter"`     /* FILTER expression, if any */
	Aggstar       bool `json:"aggstar"`       /* TRUE if argument list was really '*' */
	Aggvariadic   bool `json:"aggvariadic"`   /* true if variadic arguments have been
	 * combined into an array last argument */

	Aggkind     byte     `json:"aggkind"`     /* aggregate kind (see pg_aggregate.h) */
	Agglevelsup Index    `json:"agglevelsup"` /* > 0 if agg belongs to outer query */
	Aggsplit    AggSplit `json:"aggsplit"`    /* expected agg-splitting mode of parent Agg */
	Location    int      `json:"location"`    /* token location, or -1 if unknown */
}

func (node Aggref) MarshalJSON() ([]byte, error) {
	type AggrefMarshalAlias Aggref
	return json.Marshal(map[string]interface{}{
		"Aggref": (*AggrefMarshalAlias)(&node),
	})
}

func (node *Aggref) UnmarshalJSON(input []byte) (err error) {
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

	if fields["aggfnoid"] != nil {
		err = json.Unmarshal(fields["aggfnoid"], &node.Aggfnoid)
		if err != nil {
			return
		}
	}

	if fields["aggtype"] != nil {
		err = json.Unmarshal(fields["aggtype"], &node.Aggtype)
		if err != nil {
			return
		}
	}

	if fields["aggcollid"] != nil {
		err = json.Unmarshal(fields["aggcollid"], &node.Aggcollid)
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

	if fields["aggtranstype"] != nil {
		err = json.Unmarshal(fields["aggtranstype"], &node.Aggtranstype)
		if err != nil {
			return
		}
	}

	if fields["aggargtypes"] != nil {
		node.Aggargtypes.Items, err = UnmarshalNodeArrayJSON(fields["aggargtypes"])
		if err != nil {
			return
		}
	}

	if fields["aggdirectargs"] != nil {
		node.Aggdirectargs.Items, err = UnmarshalNodeArrayJSON(fields["aggdirectargs"])
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

	if fields["aggorder"] != nil {
		node.Aggorder.Items, err = UnmarshalNodeArrayJSON(fields["aggorder"])
		if err != nil {
			return
		}
	}

	if fields["aggdistinct"] != nil {
		node.Aggdistinct.Items, err = UnmarshalNodeArrayJSON(fields["aggdistinct"])
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

	if fields["aggstar"] != nil {
		err = json.Unmarshal(fields["aggstar"], &node.Aggstar)
		if err != nil {
			return
		}
	}

	if fields["aggvariadic"] != nil {
		err = json.Unmarshal(fields["aggvariadic"], &node.Aggvariadic)
		if err != nil {
			return
		}
	}

	if fields["aggkind"] != nil {
		var strVal string
		err = json.Unmarshal(fields["aggkind"], &strVal)
		node.Aggkind = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["agglevelsup"] != nil {
		err = json.Unmarshal(fields["agglevelsup"], &node.Agglevelsup)
		if err != nil {
			return
		}
	}

	if fields["aggsplit"] != nil {
		err = json.Unmarshal(fields["aggsplit"], &node.Aggsplit)
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
