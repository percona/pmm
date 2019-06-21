// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * WindowClause -
 *		transformed representation of WINDOW and OVER clauses
 *
 * A parsed Query's windowClause list contains these structs.  "name" is set
 * if the clause originally came from WINDOW, and is NULL if it originally
 * was an OVER clause (but note that we collapse out duplicate OVERs).
 * partitionClause and orderClause are lists of SortGroupClause structs.
 * winref is an ID number referenced by WindowFunc nodes; it must be unique
 * among the members of a Query's windowClause list.
 * When refname isn't null, the partitionClause is always copied from there;
 * the orderClause might or might not be copied (see copiedOrder); the framing
 * options are never copied, per spec.
 */
type WindowClause struct {
	Name            *string `json:"name"`            /* window name (NULL in an OVER clause) */
	Refname         *string `json:"refname"`         /* referenced window name, if any */
	PartitionClause List    `json:"partitionClause"` /* PARTITION BY list */
	OrderClause     List    `json:"orderClause"`     /* ORDER BY list */
	FrameOptions    int     `json:"frameOptions"`    /* frame_clause options, see WindowDef */
	StartOffset     Node    `json:"startOffset"`     /* expression for starting bound, if any */
	EndOffset       Node    `json:"endOffset"`       /* expression for ending bound, if any */
	Winref          Index   `json:"winref"`          /* ID referenced by window functions */
	CopiedOrder     bool    `json:"copiedOrder"`     /* did we copy orderClause from refname? */
}

func (node WindowClause) MarshalJSON() ([]byte, error) {
	type WindowClauseMarshalAlias WindowClause
	return json.Marshal(map[string]interface{}{
		"WindowClause": (*WindowClauseMarshalAlias)(&node),
	})
}

func (node *WindowClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["refname"] != nil {
		err = json.Unmarshal(fields["refname"], &node.Refname)
		if err != nil {
			return
		}
	}

	if fields["partitionClause"] != nil {
		node.PartitionClause.Items, err = UnmarshalNodeArrayJSON(fields["partitionClause"])
		if err != nil {
			return
		}
	}

	if fields["orderClause"] != nil {
		node.OrderClause.Items, err = UnmarshalNodeArrayJSON(fields["orderClause"])
		if err != nil {
			return
		}
	}

	if fields["frameOptions"] != nil {
		err = json.Unmarshal(fields["frameOptions"], &node.FrameOptions)
		if err != nil {
			return
		}
	}

	if fields["startOffset"] != nil {
		node.StartOffset, err = UnmarshalNodeJSON(fields["startOffset"])
		if err != nil {
			return
		}
	}

	if fields["endOffset"] != nil {
		node.EndOffset, err = UnmarshalNodeJSON(fields["endOffset"])
		if err != nil {
			return
		}
	}

	if fields["winref"] != nil {
		err = json.Unmarshal(fields["winref"], &node.Winref)
		if err != nil {
			return
		}
	}

	if fields["copiedOrder"] != nil {
		err = json.Unmarshal(fields["copiedOrder"], &node.CopiedOrder)
		if err != nil {
			return
		}
	}

	return
}
