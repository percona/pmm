// Auto-generated from postgres/src/include/storage/block.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * BlockId:
 *
 * this is a storage type for BlockNumber.  in other words, this type
 * is used for on-disk structures (e.g., in HeapTupleData) whereas
 * BlockNumber is the type on which calculations are performed (e.g.,
 * in access method code).
 *
 * there doesn't appear to be any reason to have separate types except
 * for the fact that BlockIds can be SHORTALIGN'd (and therefore any
 * structures that contains them, such as ItemPointerData, can also be
 * SHORTALIGN'd).  this is an important consideration for reducing the
 * space requirements of the line pointer (ItemIdData) array on each
 * page and the header of each heap or index tuple, so it doesn't seem
 * wise to change this without good reason.
 */
type BlockIdData struct {
	BiHi uint16 `json:"bi_hi"`
	BiLo uint16 `json:"bi_lo"`
}

func (node BlockIdData) MarshalJSON() ([]byte, error) {
	type BlockIdDataMarshalAlias BlockIdData
	return json.Marshal(map[string]interface{}{
		"BlockIdData": (*BlockIdDataMarshalAlias)(&node),
	})
}

func (node *BlockIdData) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["bi_hi"] != nil {
		err = json.Unmarshal(fields["bi_hi"], &node.BiHi)
		if err != nil {
			return
		}
	}

	if fields["bi_lo"] != nil {
		err = json.Unmarshal(fields["bi_lo"], &node.BiLo)
		if err != nil {
			return
		}
	}

	return
}
