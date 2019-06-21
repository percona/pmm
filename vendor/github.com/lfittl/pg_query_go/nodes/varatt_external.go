// Auto-generated from postgres/src/include/postgres.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * struct varatt_external is a traditional "TOAST pointer", that is, the
 * information needed to fetch a Datum stored out-of-line in a TOAST table.
 * The data is compressed if and only if va_extsize < va_rawsize - VARHDRSZ.
 * This struct must not contain any padding, because we sometimes compare
 * these pointers using memcmp.
 *
 * Note that this information is stored unaligned within actual tuples, so
 * you need to memcpy from the tuple into a local struct variable before
 * you can look at these fields!  (The reason we use memcmp is to avoid
 * having to do that just to detect equality of two TOAST pointers...)
 */
type varatt_external struct {
	VaRawsize    int32 `json:"va_rawsize"`    /* Original data size (includes header) */
	VaExtsize    int32 `json:"va_extsize"`    /* External saved size (doesn't) */
	VaValueid    Oid   `json:"va_valueid"`    /* Unique ID of value within TOAST table */
	VaToastrelid Oid   `json:"va_toastrelid"` /* RelID of TOAST table containing it */
}

func (node varatt_external) MarshalJSON() ([]byte, error) {
	type varatt_externalMarshalAlias varatt_external
	return json.Marshal(map[string]interface{}{
		"varatt_external": (*varatt_externalMarshalAlias)(&node),
	})
}

func (node *varatt_external) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["va_rawsize"] != nil {
		err = json.Unmarshal(fields["va_rawsize"], &node.VaRawsize)
		if err != nil {
			return
		}
	}

	if fields["va_extsize"] != nil {
		err = json.Unmarshal(fields["va_extsize"], &node.VaExtsize)
		if err != nil {
			return
		}
	}

	if fields["va_valueid"] != nil {
		err = json.Unmarshal(fields["va_valueid"], &node.VaValueid)
		if err != nil {
			return
		}
	}

	if fields["va_toastrelid"] != nil {
		err = json.Unmarshal(fields["va_toastrelid"], &node.VaToastrelid)
		if err != nil {
			return
		}
	}

	return
}
