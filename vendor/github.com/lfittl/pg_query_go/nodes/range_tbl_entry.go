// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*--------------------
 * RangeTblEntry -
 *	  A range table is a List of RangeTblEntry nodes.
 *
 *	  A range table entry may represent a plain relation, a sub-select in
 *	  FROM, or the result of a JOIN clause.  (Only explicit JOIN syntax
 *	  produces an RTE, not the implicit join resulting from multiple FROM
 *	  items.  This is because we only need the RTE to deal with SQL features
 *	  like outer joins and join-output-column aliasing.)  Other special
 *	  RTE types also exist, as indicated by RTEKind.
 *
 *	  Note that we consider RTE_RELATION to cover anything that has a pg_class
 *	  entry.  relkind distinguishes the sub-cases.
 *
 *	  alias is an Alias node representing the AS alias-clause attached to the
 *	  FROM expression, or NULL if no clause.
 *
 *	  eref is the table reference name and column reference names (either
 *	  real or aliases).  Note that system columns (OID etc) are not included
 *	  in the column list.
 *	  eref->aliasname is required to be present, and should generally be used
 *	  to identify the RTE for error messages etc.
 *
 *	  In RELATION RTEs, the colnames in both alias and eref are indexed by
 *	  physical attribute number; this means there must be colname entries for
 *	  dropped columns.  When building an RTE we insert empty strings ("") for
 *	  dropped columns.  Note however that a stored rule may have nonempty
 *	  colnames for columns dropped since the rule was created (and for that
 *	  matter the colnames might be out of date due to column renamings).
 *	  The same comments apply to FUNCTION RTEs when a function's return type
 *	  is a named composite type.
 *
 *	  In JOIN RTEs, the colnames in both alias and eref are one-to-one with
 *	  joinaliasvars entries.  A JOIN RTE will omit columns of its inputs when
 *	  those columns are known to be dropped at parse time.  Again, however,
 *	  a stored rule might contain entries for columns dropped since the rule
 *	  was created.  (This is only possible for columns not actually referenced
 *	  in the rule.)  When loading a stored rule, we replace the joinaliasvars
 *	  items for any such columns with null pointers.  (We can't simply delete
 *	  them from the joinaliasvars list, because that would affect the attnums
 *	  of Vars referencing the rest of the list.)
 *
 *	  inh is TRUE for relation references that should be expanded to include
 *	  inheritance children, if the rel has any.  This *must* be FALSE for
 *	  RTEs other than RTE_RELATION entries.
 *
 *	  inFromCl marks those range variables that are listed in the FROM clause.
 *	  It's false for RTEs that are added to a query behind the scenes, such
 *	  as the NEW and OLD variables for a rule, or the subqueries of a UNION.
 *	  This flag is not used anymore during parsing, since the parser now uses
 *	  a separate "namespace" data structure to control visibility, but it is
 *	  needed by ruleutils.c to determine whether RTEs should be shown in
 *	  decompiled queries.
 *
 *	  requiredPerms and checkAsUser specify run-time access permissions
 *	  checks to be performed at query startup.  The user must have *all*
 *	  of the permissions that are OR'd together in requiredPerms (zero
 *	  indicates no permissions checking).  If checkAsUser is not zero,
 *	  then do the permissions checks using the access rights of that user,
 *	  not the current effective user ID.  (This allows rules to act as
 *	  setuid gateways.)  Permissions checks only apply to RELATION RTEs.
 *
 *	  For SELECT/INSERT/UPDATE permissions, if the user doesn't have
 *	  table-wide permissions then it is sufficient to have the permissions
 *	  on all columns identified in selectedCols (for SELECT) and/or
 *	  insertedCols and/or updatedCols (INSERT with ON CONFLICT DO UPDATE may
 *	  have all 3).  selectedCols, insertedCols and updatedCols are bitmapsets,
 *	  which cannot have negative integer members, so we subtract
 *	  FirstLowInvalidHeapAttributeNumber from column numbers before storing
 *	  them in these fields.  A whole-row Var reference is represented by
 *	  setting the bit for InvalidAttrNumber.
 *
 *	  securityQuals is a list of security barrier quals (boolean expressions),
 *	  to be tested in the listed order before returning a row from the
 *	  relation.  It is always NIL in parser output.  Entries are added by the
 *	  rewriter to implement security-barrier views and/or row-level security.
 *	  Note that the planner turns each boolean expression into an implicitly
 *	  AND'ed sublist, as is its usual habit with qualification expressions.
 *--------------------
 */
type RangeTblEntry struct {
	Rtekind RTEKind `json:"rtekind"` /* see above */

	/*
	 * XXX the fields applicable to only some rte kinds should be merged into
	 * a union.  I didn't do this yet because the diffs would impact a lot of
	 * code that is being actively worked on.  FIXME someday.
	 */

	/*
	 * Fields valid for a plain relation RTE (else zero):
	 *
	 * As a special case, RTE_NAMEDTUPLESTORE can also set relid to indicate
	 * that the tuple format of the tuplestore is the same as the referenced
	 * relation.  This allows plans referencing AFTER trigger transition
	 * tables to be invalidated if the underlying table is altered.
	 */
	Relid       Oid                `json:"relid"`       /* OID of the relation */
	Relkind     byte               `json:"relkind"`     /* relation kind (see pg_class.relkind) */
	Tablesample *TableSampleClause `json:"tablesample"` /* sampling info, or NULL */

	/*
	 * Fields valid for a subquery RTE (else NULL):
	 */
	Subquery        *Query `json:"subquery"`         /* the sub-query */
	SecurityBarrier bool   `json:"security_barrier"` /* is from security_barrier view? */

	/*
	 * Fields valid for a join RTE (else NULL/zero):
	 *
	 * joinaliasvars is a list of (usually) Vars corresponding to the columns
	 * of the join result.  An alias Var referencing column K of the join
	 * result can be replaced by the K'th element of joinaliasvars --- but to
	 * simplify the task of reverse-listing aliases correctly, we do not do
	 * that until planning time.  In detail: an element of joinaliasvars can
	 * be a Var of one of the join's input relations, or such a Var with an
	 * implicit coercion to the join's output column type, or a COALESCE
	 * expression containing the two input column Vars (possibly coerced).
	 * Within a Query loaded from a stored rule, it is also possible for
	 * joinaliasvars items to be null pointers, which are placeholders for
	 * (necessarily unreferenced) columns dropped since the rule was made.
	 * Also, once planning begins, joinaliasvars items can be almost anything,
	 * as a result of subquery-flattening substitutions.
	 */
	Jointype      JoinType `json:"jointype"`      /* type of join */
	Joinaliasvars List     `json:"joinaliasvars"` /* list of alias-var expansions */

	/*
	 * Fields valid for a function RTE (else NIL/zero):
	 *
	 * When funcordinality is true, the eref->colnames list includes an alias
	 * for the ordinality column.  The ordinality column is otherwise
	 * implicit, and must be accounted for "by hand" in places such as
	 * expandRTE().
	 */
	Functions      List `json:"functions"`      /* list of RangeTblFunction nodes */
	Funcordinality bool `json:"funcordinality"` /* is this called WITH ORDINALITY? */

	/*
	 * Fields valid for a TableFunc RTE (else NULL):
	 */
	Tablefunc *TableFunc `json:"tablefunc"`

	/*
	 * Fields valid for a values RTE (else NIL):
	 */
	ValuesLists List `json:"values_lists"` /* list of expression lists */

	/*
	 * Fields valid for a CTE RTE (else NULL/zero):
	 */
	Ctename       *string `json:"ctename"`        /* name of the WITH list item */
	Ctelevelsup   Index   `json:"ctelevelsup"`    /* number of query levels up */
	SelfReference bool    `json:"self_reference"` /* is this a recursive self-reference? */

	/*
	 * Fields valid for table functions, values, CTE and ENR RTEs (else NIL):
	 *
	 * We need these for CTE RTEs so that the types of self-referential
	 * columns are well-defined.  For VALUES RTEs, storing these explicitly
	 * saves having to re-determine the info by scanning the values_lists. For
	 * ENRs, we store the types explicitly here (we could get the information
	 * from the catalogs if 'relid' was supplied, but we'd still need these
	 * for TupleDesc-based ENRs, so we might as well always store the type
	 * info here).
	 */
	Coltypes      List `json:"coltypes"`      /* OID list of column type OIDs */
	Coltypmods    List `json:"coltypmods"`    /* integer list of column typmods */
	Colcollations List `json:"colcollations"` /* OID list of column collation OIDs */

	/*
	 * Fields valid for ENR RTEs (else NULL/zero):
	 */
	Enrname   *string `json:"enrname"`   /* name of ephemeral named relation */
	Enrtuples float64 `json:"enrtuples"` /* estimated or actual from caller */

	/*
	 * Fields valid in all RTEs:
	 */
	Alias         *Alias   `json:"alias"`         /* user-written alias clause, if any */
	Eref          *Alias   `json:"eref"`          /* expanded reference names */
	Lateral       bool     `json:"lateral"`       /* subquery, function, or values is LATERAL? */
	Inh           bool     `json:"inh"`           /* inheritance requested? */
	InFromCl      bool     `json:"inFromCl"`      /* present in FROM clause? */
	RequiredPerms AclMode  `json:"requiredPerms"` /* bitmask of required access permissions */
	CheckAsUser   Oid      `json:"checkAsUser"`   /* if valid, check access as this role */
	SelectedCols  []uint32 `json:"selectedCols"`  /* columns needing SELECT permission */
	InsertedCols  []uint32 `json:"insertedCols"`  /* columns needing INSERT permission */
	UpdatedCols   []uint32 `json:"updatedCols"`   /* columns needing UPDATE permission */
	SecurityQuals List     `json:"securityQuals"` /* security barrier quals to apply, if any */
}

func (node RangeTblEntry) MarshalJSON() ([]byte, error) {
	type RangeTblEntryMarshalAlias RangeTblEntry
	return json.Marshal(map[string]interface{}{
		"RangeTblEntry": (*RangeTblEntryMarshalAlias)(&node),
	})
}

func (node *RangeTblEntry) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["rtekind"] != nil {
		err = json.Unmarshal(fields["rtekind"], &node.Rtekind)
		if err != nil {
			return
		}
	}

	if fields["relid"] != nil {
		err = json.Unmarshal(fields["relid"], &node.Relid)
		if err != nil {
			return
		}
	}

	if fields["relkind"] != nil {
		var strVal string
		err = json.Unmarshal(fields["relkind"], &strVal)
		node.Relkind = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["tablesample"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["tablesample"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TableSampleClause)
			node.Tablesample = &val
		}
	}

	if fields["subquery"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["subquery"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(Query)
			node.Subquery = &val
		}
	}

	if fields["security_barrier"] != nil {
		err = json.Unmarshal(fields["security_barrier"], &node.SecurityBarrier)
		if err != nil {
			return
		}
	}

	if fields["jointype"] != nil {
		err = json.Unmarshal(fields["jointype"], &node.Jointype)
		if err != nil {
			return
		}
	}

	if fields["joinaliasvars"] != nil {
		node.Joinaliasvars.Items, err = UnmarshalNodeArrayJSON(fields["joinaliasvars"])
		if err != nil {
			return
		}
	}

	if fields["functions"] != nil {
		node.Functions.Items, err = UnmarshalNodeArrayJSON(fields["functions"])
		if err != nil {
			return
		}
	}

	if fields["funcordinality"] != nil {
		err = json.Unmarshal(fields["funcordinality"], &node.Funcordinality)
		if err != nil {
			return
		}
	}

	if fields["tablefunc"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["tablefunc"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TableFunc)
			node.Tablefunc = &val
		}
	}

	if fields["values_lists"] != nil {
		node.ValuesLists.Items, err = UnmarshalNodeArrayJSON(fields["values_lists"])
		if err != nil {
			return
		}
	}

	if fields["ctename"] != nil {
		err = json.Unmarshal(fields["ctename"], &node.Ctename)
		if err != nil {
			return
		}
	}

	if fields["ctelevelsup"] != nil {
		err = json.Unmarshal(fields["ctelevelsup"], &node.Ctelevelsup)
		if err != nil {
			return
		}
	}

	if fields["self_reference"] != nil {
		err = json.Unmarshal(fields["self_reference"], &node.SelfReference)
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

	if fields["enrname"] != nil {
		err = json.Unmarshal(fields["enrname"], &node.Enrname)
		if err != nil {
			return
		}
	}

	if fields["enrtuples"] != nil {
		err = json.Unmarshal(fields["enrtuples"], &node.Enrtuples)
		if err != nil {
			return
		}
	}

	if fields["alias"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["alias"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(Alias)
			node.Alias = &val
		}
	}

	if fields["eref"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["eref"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(Alias)
			node.Eref = &val
		}
	}

	if fields["lateral"] != nil {
		err = json.Unmarshal(fields["lateral"], &node.Lateral)
		if err != nil {
			return
		}
	}

	if fields["inh"] != nil {
		err = json.Unmarshal(fields["inh"], &node.Inh)
		if err != nil {
			return
		}
	}

	if fields["inFromCl"] != nil {
		err = json.Unmarshal(fields["inFromCl"], &node.InFromCl)
		if err != nil {
			return
		}
	}

	if fields["requiredPerms"] != nil {
		err = json.Unmarshal(fields["requiredPerms"], &node.RequiredPerms)
		if err != nil {
			return
		}
	}

	if fields["checkAsUser"] != nil {
		err = json.Unmarshal(fields["checkAsUser"], &node.CheckAsUser)
		if err != nil {
			return
		}
	}

	if fields["selectedCols"] != nil {
		err = json.Unmarshal(fields["selectedCols"], &node.SelectedCols)
		if err != nil {
			return
		}
	}

	if fields["insertedCols"] != nil {
		err = json.Unmarshal(fields["insertedCols"], &node.InsertedCols)
		if err != nil {
			return
		}
	}

	if fields["updatedCols"] != nil {
		err = json.Unmarshal(fields["updatedCols"], &node.UpdatedCols)
		if err != nil {
			return
		}
	}

	if fields["securityQuals"] != nil {
		node.SecurityQuals.Items, err = UnmarshalNodeArrayJSON(fields["securityQuals"])
		if err != nil {
			return
		}
	}

	return
}
