// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* Foreign key matchtype codes */
type Constraint struct {
	Contype ConstrType `json:"contype"` /* see above */

	/* Fields used for most/all constraint types: */
	Conname      *string `json:"conname"`      /* Constraint name, or NULL if unnamed */
	Deferrable   bool    `json:"deferrable"`   /* DEFERRABLE? */
	Initdeferred bool    `json:"initdeferred"` /* INITIALLY DEFERRED? */
	Location     int     `json:"location"`     /* token location, or -1 if unknown */

	/* Fields used for constraints with expressions (CHECK and DEFAULT): */
	IsNoInherit   bool    `json:"is_no_inherit"` /* is constraint non-inheritable? */
	RawExpr       Node    `json:"raw_expr"`      /* expr, as untransformed parse tree */
	CookedExpr    *string `json:"cooked_expr"`   /* expr, as nodeToString representation */
	GeneratedWhen byte    `json:"generated_when"`

	/* Fields used for unique constraints (UNIQUE and PRIMARY KEY): */
	Keys List `json:"keys"` /* String nodes naming referenced column(s) */

	/* Fields used for EXCLUSION constraints: */
	Exclusions List `json:"exclusions"` /* list of (IndexElem, operator name) pairs */

	/* Fields used for index constraints (UNIQUE, PRIMARY KEY, EXCLUSION): */
	Options    List    `json:"options"`    /* options from WITH clause */
	Indexname  *string `json:"indexname"`  /* existing index to use; otherwise NULL */
	Indexspace *string `json:"indexspace"` /* index tablespace; NULL for default */

	/* These could be, but currently are not, used for UNIQUE/PKEY: */
	AccessMethod *string `json:"access_method"` /* index access method; NULL for default */
	WhereClause  Node    `json:"where_clause"`  /* partial index predicate */

	/* Fields used for FOREIGN KEY constraints: */
	Pktable       *RangeVar `json:"pktable"`         /* Primary key table */
	FkAttrs       List      `json:"fk_attrs"`        /* Attributes of foreign key */
	PkAttrs       List      `json:"pk_attrs"`        /* Corresponding attrs in PK table */
	FkMatchtype   byte      `json:"fk_matchtype"`    /* FULL, PARTIAL, SIMPLE */
	FkUpdAction   byte      `json:"fk_upd_action"`   /* ON UPDATE action */
	FkDelAction   byte      `json:"fk_del_action"`   /* ON DELETE action */
	OldConpfeqop  List      `json:"old_conpfeqop"`   /* pg_constraint.conpfeqop of my former self */
	OldPktableOid Oid       `json:"old_pktable_oid"` /* pg_constraint.confrelid of my former
	 * self */

	/* Fields used for constraints that allow a NOT VALID specification */
	SkipValidation bool `json:"skip_validation"` /* skip validation of existing rows? */
	InitiallyValid bool `json:"initially_valid"` /* mark the new constraint as valid? */
}

func (node Constraint) MarshalJSON() ([]byte, error) {
	type ConstraintMarshalAlias Constraint
	return json.Marshal(map[string]interface{}{
		"Constraint": (*ConstraintMarshalAlias)(&node),
	})
}

func (node *Constraint) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["contype"] != nil {
		err = json.Unmarshal(fields["contype"], &node.Contype)
		if err != nil {
			return
		}
	}

	if fields["conname"] != nil {
		err = json.Unmarshal(fields["conname"], &node.Conname)
		if err != nil {
			return
		}
	}

	if fields["deferrable"] != nil {
		err = json.Unmarshal(fields["deferrable"], &node.Deferrable)
		if err != nil {
			return
		}
	}

	if fields["initdeferred"] != nil {
		err = json.Unmarshal(fields["initdeferred"], &node.Initdeferred)
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

	if fields["is_no_inherit"] != nil {
		err = json.Unmarshal(fields["is_no_inherit"], &node.IsNoInherit)
		if err != nil {
			return
		}
	}

	if fields["raw_expr"] != nil {
		node.RawExpr, err = UnmarshalNodeJSON(fields["raw_expr"])
		if err != nil {
			return
		}
	}

	if fields["cooked_expr"] != nil {
		err = json.Unmarshal(fields["cooked_expr"], &node.CookedExpr)
		if err != nil {
			return
		}
	}

	if fields["generated_when"] != nil {
		var strVal string
		err = json.Unmarshal(fields["generated_when"], &strVal)
		node.GeneratedWhen = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["keys"] != nil {
		node.Keys.Items, err = UnmarshalNodeArrayJSON(fields["keys"])
		if err != nil {
			return
		}
	}

	if fields["exclusions"] != nil {
		node.Exclusions.Items, err = UnmarshalNodeArrayJSON(fields["exclusions"])
		if err != nil {
			return
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	if fields["indexname"] != nil {
		err = json.Unmarshal(fields["indexname"], &node.Indexname)
		if err != nil {
			return
		}
	}

	if fields["indexspace"] != nil {
		err = json.Unmarshal(fields["indexspace"], &node.Indexspace)
		if err != nil {
			return
		}
	}

	if fields["access_method"] != nil {
		err = json.Unmarshal(fields["access_method"], &node.AccessMethod)
		if err != nil {
			return
		}
	}

	if fields["where_clause"] != nil {
		node.WhereClause, err = UnmarshalNodeJSON(fields["where_clause"])
		if err != nil {
			return
		}
	}

	if fields["pktable"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["pktable"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Pktable = &val
		}
	}

	if fields["fk_attrs"] != nil {
		node.FkAttrs.Items, err = UnmarshalNodeArrayJSON(fields["fk_attrs"])
		if err != nil {
			return
		}
	}

	if fields["pk_attrs"] != nil {
		node.PkAttrs.Items, err = UnmarshalNodeArrayJSON(fields["pk_attrs"])
		if err != nil {
			return
		}
	}

	if fields["fk_matchtype"] != nil {
		var strVal string
		err = json.Unmarshal(fields["fk_matchtype"], &strVal)
		node.FkMatchtype = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["fk_upd_action"] != nil {
		var strVal string
		err = json.Unmarshal(fields["fk_upd_action"], &strVal)
		node.FkUpdAction = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["fk_del_action"] != nil {
		var strVal string
		err = json.Unmarshal(fields["fk_del_action"], &strVal)
		node.FkDelAction = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["old_conpfeqop"] != nil {
		node.OldConpfeqop.Items, err = UnmarshalNodeArrayJSON(fields["old_conpfeqop"])
		if err != nil {
			return
		}
	}

	if fields["old_pktable_oid"] != nil {
		err = json.Unmarshal(fields["old_pktable_oid"], &node.OldPktableOid)
		if err != nil {
			return
		}
	}

	if fields["skip_validation"] != nil {
		err = json.Unmarshal(fields["skip_validation"], &node.SkipValidation)
		if err != nil {
			return
		}
	}

	if fields["initially_valid"] != nil {
		err = json.Unmarshal(fields["initially_valid"], &node.InitiallyValid)
		if err != nil {
			return
		}
	}

	return
}
