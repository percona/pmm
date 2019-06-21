// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------
 * OnConflictExpr - represents an ON CONFLICT DO ... expression
 *
 * The optimizer requires a list of inference elements, and optionally a WHERE
 * clause to infer a unique index.  The unique index (or, occasionally,
 * indexes) inferred are used to arbitrate whether or not the alternative ON
 * CONFLICT path is taken.
 *----------
 */
type OnConflictExpr struct {
	Action OnConflictAction `json:"action"` /* DO NOTHING or UPDATE? */

	/* Arbiter */
	ArbiterElems List `json:"arbiterElems"` /* unique index arbiter list (of
	 * InferenceElem's) */

	ArbiterWhere Node `json:"arbiterWhere"` /* unique index arbiter WHERE clause */
	Constraint   Oid  `json:"constraint"`   /* pg_constraint OID for arbiter */

	/* ON CONFLICT UPDATE */
	OnConflictSet   List `json:"onConflictSet"`   /* List of ON CONFLICT SET TargetEntrys */
	OnConflictWhere Node `json:"onConflictWhere"` /* qualifiers to restrict UPDATE to */
	ExclRelIndex    int  `json:"exclRelIndex"`    /* RT index of 'excluded' relation */
	ExclRelTlist    List `json:"exclRelTlist"`    /* tlist of the EXCLUDED pseudo relation */
}

func (node OnConflictExpr) MarshalJSON() ([]byte, error) {
	type OnConflictExprMarshalAlias OnConflictExpr
	return json.Marshal(map[string]interface{}{
		"OnConflictExpr": (*OnConflictExprMarshalAlias)(&node),
	})
}

func (node *OnConflictExpr) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["action"] != nil {
		err = json.Unmarshal(fields["action"], &node.Action)
		if err != nil {
			return
		}
	}

	if fields["arbiterElems"] != nil {
		node.ArbiterElems.Items, err = UnmarshalNodeArrayJSON(fields["arbiterElems"])
		if err != nil {
			return
		}
	}

	if fields["arbiterWhere"] != nil {
		node.ArbiterWhere, err = UnmarshalNodeJSON(fields["arbiterWhere"])
		if err != nil {
			return
		}
	}

	if fields["constraint"] != nil {
		err = json.Unmarshal(fields["constraint"], &node.Constraint)
		if err != nil {
			return
		}
	}

	if fields["onConflictSet"] != nil {
		node.OnConflictSet.Items, err = UnmarshalNodeArrayJSON(fields["onConflictSet"])
		if err != nil {
			return
		}
	}

	if fields["onConflictWhere"] != nil {
		node.OnConflictWhere, err = UnmarshalNodeJSON(fields["onConflictWhere"])
		if err != nil {
			return
		}
	}

	if fields["exclRelIndex"] != nil {
		err = json.Unmarshal(fields["exclRelIndex"], &node.ExclRelIndex)
		if err != nil {
			return
		}
	}

	if fields["exclRelTlist"] != nil {
		node.ExclRelTlist.Items, err = UnmarshalNodeArrayJSON(fields["exclRelTlist"])
		if err != nil {
			return
		}
	}

	return
}
