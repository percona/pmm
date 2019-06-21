// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * ColumnDef - column definition (used in various creates)
 *
 * If the column has a default value, we may have the value expression
 * in either "raw" form (an untransformed parse tree) or "cooked" form
 * (a post-parse-analysis, executable expression tree), depending on
 * how this ColumnDef node was created (by parsing, or by inheritance
 * from an existing relation).  We should never have both in the same node!
 *
 * Similarly, we may have a COLLATE specification in either raw form
 * (represented as a CollateClause with arg==NULL) or cooked form
 * (the collation's OID).
 *
 * The constraints list may contain a CONSTR_DEFAULT item in a raw
 * parsetree produced by gram.y, but transformCreateStmt will remove
 * the item and set raw_default instead.  CONSTR_DEFAULT items
 * should not appear in any subsequent processing.
 */
type ColumnDef struct {
	Colname       *string        `json:"colname"`        /* name of column */
	TypeName      *TypeName      `json:"typeName"`       /* type of column */
	Inhcount      int            `json:"inhcount"`       /* number of times column is inherited */
	IsLocal       bool           `json:"is_local"`       /* column has local (non-inherited) def'n */
	IsNotNull     bool           `json:"is_not_null"`    /* NOT NULL constraint specified? */
	IsFromType    bool           `json:"is_from_type"`   /* column definition came from table type */
	IsFromParent  bool           `json:"is_from_parent"` /* column def came from partition parent */
	Storage       byte           `json:"storage"`        /* attstorage setting, or 0 for default */
	RawDefault    Node           `json:"raw_default"`    /* default value (untransformed parse tree) */
	CookedDefault Node           `json:"cooked_default"` /* default value (transformed expr tree) */
	Identity      byte           `json:"identity"`       /* attidentity setting */
	CollClause    *CollateClause `json:"collClause"`     /* untransformed COLLATE spec, if any */
	CollOid       Oid            `json:"collOid"`        /* collation OID (InvalidOid if not set) */
	Constraints   List           `json:"constraints"`    /* other constraints on column */
	Fdwoptions    List           `json:"fdwoptions"`     /* per-column FDW options */
	Location      int            `json:"location"`       /* parse location, or -1 if none/unknown */
}

func (node ColumnDef) MarshalJSON() ([]byte, error) {
	type ColumnDefMarshalAlias ColumnDef
	return json.Marshal(map[string]interface{}{
		"ColumnDef": (*ColumnDefMarshalAlias)(&node),
	})
}

func (node *ColumnDef) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["colname"] != nil {
		err = json.Unmarshal(fields["colname"], &node.Colname)
		if err != nil {
			return
		}
	}

	if fields["typeName"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["typeName"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.TypeName = &val
		}
	}

	if fields["inhcount"] != nil {
		err = json.Unmarshal(fields["inhcount"], &node.Inhcount)
		if err != nil {
			return
		}
	}

	if fields["is_local"] != nil {
		err = json.Unmarshal(fields["is_local"], &node.IsLocal)
		if err != nil {
			return
		}
	}

	if fields["is_not_null"] != nil {
		err = json.Unmarshal(fields["is_not_null"], &node.IsNotNull)
		if err != nil {
			return
		}
	}

	if fields["is_from_type"] != nil {
		err = json.Unmarshal(fields["is_from_type"], &node.IsFromType)
		if err != nil {
			return
		}
	}

	if fields["is_from_parent"] != nil {
		err = json.Unmarshal(fields["is_from_parent"], &node.IsFromParent)
		if err != nil {
			return
		}
	}

	if fields["storage"] != nil {
		var strVal string
		err = json.Unmarshal(fields["storage"], &strVal)
		node.Storage = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["raw_default"] != nil {
		node.RawDefault, err = UnmarshalNodeJSON(fields["raw_default"])
		if err != nil {
			return
		}
	}

	if fields["cooked_default"] != nil {
		node.CookedDefault, err = UnmarshalNodeJSON(fields["cooked_default"])
		if err != nil {
			return
		}
	}

	if fields["identity"] != nil {
		var strVal string
		err = json.Unmarshal(fields["identity"], &strVal)
		node.Identity = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["collClause"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["collClause"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(CollateClause)
			node.CollClause = &val
		}
	}

	if fields["collOid"] != nil {
		err = json.Unmarshal(fields["collOid"], &node.CollOid)
		if err != nil {
			return
		}
	}

	if fields["constraints"] != nil {
		node.Constraints.Items, err = UnmarshalNodeArrayJSON(fields["constraints"])
		if err != nil {
			return
		}
	}

	if fields["fdwoptions"] != nil {
		node.Fdwoptions.Items, err = UnmarshalNodeArrayJSON(fields["fdwoptions"])
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
