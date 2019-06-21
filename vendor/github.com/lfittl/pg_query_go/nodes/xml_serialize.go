// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * XMLSERIALIZE (in raw parse tree only)
 */
type XmlSerialize struct {
	Xmloption XmlOptionType `json:"xmloption"` /* DOCUMENT or CONTENT */
	Expr      Node          `json:"expr"`
	TypeName  *TypeName     `json:"typeName"`
	Location  int           `json:"location"` /* token location, or -1 if unknown */
}

func (node XmlSerialize) MarshalJSON() ([]byte, error) {
	type XmlSerializeMarshalAlias XmlSerialize
	return json.Marshal(map[string]interface{}{
		"XmlSerialize": (*XmlSerializeMarshalAlias)(&node),
	})
}

func (node *XmlSerialize) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xmloption"] != nil {
		err = json.Unmarshal(fields["xmloption"], &node.Xmloption)
		if err != nil {
			return
		}
	}

	if fields["expr"] != nil {
		node.Expr, err = UnmarshalNodeJSON(fields["expr"])
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

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
		if err != nil {
			return
		}
	}

	return
}
