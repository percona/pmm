// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Index Statement
 *
 * This represents creation of an index and/or an associated constraint.
 * If isconstraint is true, we should create a pg_constraint entry along
 * with the index.  But if indexOid isn't InvalidOid, we are not creating an
 * index, just a UNIQUE/PKEY constraint using an existing index.  isconstraint
 * must always be true in this case, and the fields describing the index
 * properties are empty.
 * ----------------------
 */
type IndexStmt struct {
	Idxname        *string   `json:"idxname"`        /* name of new index, or NULL for default */
	Relation       *RangeVar `json:"relation"`       /* relation to build index on */
	AccessMethod   *string   `json:"accessMethod"`   /* name of access method (eg. btree) */
	TableSpace     *string   `json:"tableSpace"`     /* tablespace, or NULL for default */
	IndexParams    List      `json:"indexParams"`    /* columns to index: a list of IndexElem */
	Options        List      `json:"options"`        /* WITH clause options: a list of DefElem */
	WhereClause    Node      `json:"whereClause"`    /* qualification (partial-index predicate) */
	ExcludeOpNames List      `json:"excludeOpNames"` /* exclusion operator names, or NIL if none */
	Idxcomment     *string   `json:"idxcomment"`     /* comment to apply to index, or NULL */
	IndexOid       Oid       `json:"indexOid"`       /* OID of an existing index, if any */
	OldNode        Oid       `json:"oldNode"`        /* relfilenode of existing storage, if any */
	Unique         bool      `json:"unique"`         /* is index unique? */
	Primary        bool      `json:"primary"`        /* is index a primary key? */
	Isconstraint   bool      `json:"isconstraint"`   /* is it for a pkey/unique constraint? */
	Deferrable     bool      `json:"deferrable"`     /* is the constraint DEFERRABLE? */
	Initdeferred   bool      `json:"initdeferred"`   /* is the constraint INITIALLY DEFERRED? */
	Transformed    bool      `json:"transformed"`    /* true when transformIndexStmt is finished */
	Concurrent     bool      `json:"concurrent"`     /* should this be a concurrent index build? */
	IfNotExists    bool      `json:"if_not_exists"`  /* just do nothing if index already exists? */
}

func (node IndexStmt) MarshalJSON() ([]byte, error) {
	type IndexStmtMarshalAlias IndexStmt
	return json.Marshal(map[string]interface{}{
		"IndexStmt": (*IndexStmtMarshalAlias)(&node),
	})
}

func (node *IndexStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["idxname"] != nil {
		err = json.Unmarshal(fields["idxname"], &node.Idxname)
		if err != nil {
			return
		}
	}

	if fields["relation"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["relation"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Relation = &val
		}
	}

	if fields["accessMethod"] != nil {
		err = json.Unmarshal(fields["accessMethod"], &node.AccessMethod)
		if err != nil {
			return
		}
	}

	if fields["tableSpace"] != nil {
		err = json.Unmarshal(fields["tableSpace"], &node.TableSpace)
		if err != nil {
			return
		}
	}

	if fields["indexParams"] != nil {
		node.IndexParams.Items, err = UnmarshalNodeArrayJSON(fields["indexParams"])
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

	if fields["whereClause"] != nil {
		node.WhereClause, err = UnmarshalNodeJSON(fields["whereClause"])
		if err != nil {
			return
		}
	}

	if fields["excludeOpNames"] != nil {
		node.ExcludeOpNames.Items, err = UnmarshalNodeArrayJSON(fields["excludeOpNames"])
		if err != nil {
			return
		}
	}

	if fields["idxcomment"] != nil {
		err = json.Unmarshal(fields["idxcomment"], &node.Idxcomment)
		if err != nil {
			return
		}
	}

	if fields["indexOid"] != nil {
		err = json.Unmarshal(fields["indexOid"], &node.IndexOid)
		if err != nil {
			return
		}
	}

	if fields["oldNode"] != nil {
		err = json.Unmarshal(fields["oldNode"], &node.OldNode)
		if err != nil {
			return
		}
	}

	if fields["unique"] != nil {
		err = json.Unmarshal(fields["unique"], &node.Unique)
		if err != nil {
			return
		}
	}

	if fields["primary"] != nil {
		err = json.Unmarshal(fields["primary"], &node.Primary)
		if err != nil {
			return
		}
	}

	if fields["isconstraint"] != nil {
		err = json.Unmarshal(fields["isconstraint"], &node.Isconstraint)
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

	if fields["transformed"] != nil {
		err = json.Unmarshal(fields["transformed"], &node.Transformed)
		if err != nil {
			return
		}
	}

	if fields["concurrent"] != nil {
		err = json.Unmarshal(fields["concurrent"], &node.Concurrent)
		if err != nil {
			return
		}
	}

	if fields["if_not_exists"] != nil {
		err = json.Unmarshal(fields["if_not_exists"], &node.IfNotExists)
		if err != nil {
			return
		}
	}

	return
}
