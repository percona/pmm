// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * IntoClause - target information for SELECT INTO, CREATE TABLE AS, and
 * CREATE MATERIALIZED VIEW
 *
 * For CREATE MATERIALIZED VIEW, viewQuery is the parsed-but-not-rewritten
 * SELECT Query for the view; otherwise it's NULL.  (Although it's actually
 * Query*, we declare it as Node* to avoid a forward reference.)
 */
type IntoClause struct {
	Rel            *RangeVar      `json:"rel"`            /* target relation name */
	ColNames       List           `json:"colNames"`       /* column names to assign, or NIL */
	Options        List           `json:"options"`        /* options from WITH clause */
	OnCommit       OnCommitAction `json:"onCommit"`       /* what do we do at COMMIT? */
	TableSpaceName *string        `json:"tableSpaceName"` /* table space to use, or NULL */
	ViewQuery      Node           `json:"viewQuery"`      /* materialized view's SELECT query */
	SkipData       bool           `json:"skipData"`       /* true for WITH NO DATA */
}

func (node IntoClause) MarshalJSON() ([]byte, error) {
	type IntoClauseMarshalAlias IntoClause
	return json.Marshal(map[string]interface{}{
		"IntoClause": (*IntoClauseMarshalAlias)(&node),
	})
}

func (node *IntoClause) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["rel"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["rel"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Rel = &val
		}
	}

	if fields["colNames"] != nil {
		node.ColNames.Items, err = UnmarshalNodeArrayJSON(fields["colNames"])
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

	if fields["onCommit"] != nil {
		err = json.Unmarshal(fields["onCommit"], &node.OnCommit)
		if err != nil {
			return
		}
	}

	if fields["tableSpaceName"] != nil {
		err = json.Unmarshal(fields["tableSpaceName"], &node.TableSpaceName)
		if err != nil {
			return
		}
	}

	if fields["viewQuery"] != nil {
		node.ViewQuery, err = UnmarshalNodeJSON(fields["viewQuery"])
		if err != nil {
			return
		}
	}

	if fields["skipData"] != nil {
		err = json.Unmarshal(fields["skipData"], &node.SkipData)
		if err != nil {
			return
		}
	}

	return
}
