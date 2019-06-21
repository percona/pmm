// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * WindowDef - raw representation of WINDOW and OVER clauses
 *
 * For entries in a WINDOW list, "name" is the window name being defined.
 * For OVER clauses, we use "name" for the "OVER window" syntax, or "refname"
 * for the "OVER (window)" syntax, which is subtly different --- the latter
 * implies overriding the window frame clause.
 */
type WindowDef struct {
	Name            *string `json:"name"`            /* window's own name */
	Refname         *string `json:"refname"`         /* referenced window name, if any */
	PartitionClause List    `json:"partitionClause"` /* PARTITION BY expression list */
	OrderClause     List    `json:"orderClause"`     /* ORDER BY (list of SortBy) */
	FrameOptions    int     `json:"frameOptions"`    /* frame_clause options, see below */
	StartOffset     Node    `json:"startOffset"`     /* expression for starting bound, if any */
	EndOffset       Node    `json:"endOffset"`       /* expression for ending bound, if any */
	Location        int     `json:"location"`        /* parse location, or -1 if none/unknown */
}

func (node WindowDef) MarshalJSON() ([]byte, error) {
	type WindowDefMarshalAlias WindowDef
	return json.Marshal(map[string]interface{}{
		"WindowDef": (*WindowDefMarshalAlias)(&node),
	})
}

func (node *WindowDef) UnmarshalJSON(input []byte) (err error) {
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

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
		if err != nil {
			return
		}
	}

	return
}
