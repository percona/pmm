package pg_query

import "encoding/json"

func UnmarshalNodePtrJSON(input json.RawMessage) (nodePtr *Node, err error) {
	if input == nil {
		return
	}

	node, err := UnmarshalNodeJSON(input)
	if err != nil {
		return
	}

	nodePtr = &node
	return
}

func UnmarshalNodeArrayJSON(input json.RawMessage) (nodes []Node, err error) {
	var items []json.RawMessage

	err = json.Unmarshal(input, &items)
	if err != nil {
		return
	}

	for _, itemJSON := range items {
		var node Node
		node, err = UnmarshalNodeJSON(itemJSON)
		if err != nil {
			return
		}

		nodes = append(nodes, node)
	}

	return
}

func UnmarshalNodeArrayArrayJSON(input json.RawMessage) (nodeLists [][]Node, err error) {
	var items []json.RawMessage

	err = json.Unmarshal(input, &items)
	if err != nil {
		return
	}

	for _, itemJSON := range items {
		var nodeList []Node
		nodeList, err = UnmarshalNodeArrayJSON(itemJSON)
		if err != nil {
			return
		}

		nodeLists = append(nodeLists, nodeList)
	}

	return
}
