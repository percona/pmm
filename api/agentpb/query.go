package agentpb

import (
	fmt "fmt"

	"github.com/golang/protobuf/proto"
)

//go-sumtype:decl isQueryActionValue_Kind

// MarshalActionQueryResult returns serialized form of query Action result.
//
// It supports the following types:
//  * untyped nil;
//  * bool;
//  * int, int8, int16, int32, int64;
//  * uint, uint8, uint16, uint32, uint64;
//  * float32, float64;
//  * string;
//  * time.Time;
//  * []T for any T, including other slices and maps;
//  * map[string]T for any T, including other slices and maps.
func MarshalActionQueryResult(data []map[string]interface{}) ([]byte, error) {
	res := QueryActionResult{
		Res: make([]*QueryActionMap, len(data)),
	}

	for i, row := range data {
		m := QueryActionMap{
			Map: make(map[string]*QueryActionValue, len(row)),
		}

		for column, value := range row {
			var mv QueryActionValue

			switch v := value.(type) {
			case nil:
				mv = QueryActionValue{Kind: &QueryActionValue_Nil{Nil: true}}

			case bool:
				mv = QueryActionValue{Kind: &QueryActionValue_Bool{Bool: v}}

			case int:
				mv = QueryActionValue{Kind: &QueryActionValue_Int64{Int64: int64(v)}}
			case int8:
				mv = QueryActionValue{Kind: &QueryActionValue_Int64{Int64: int64(v)}}
			case int16:
				mv = QueryActionValue{Kind: &QueryActionValue_Int64{Int64: int64(v)}}
			case int32:
				mv = QueryActionValue{Kind: &QueryActionValue_Int64{Int64: int64(v)}}
			case int64:
				mv = QueryActionValue{Kind: &QueryActionValue_Int64{Int64: v}}

			case uint:
				mv = QueryActionValue{Kind: &QueryActionValue_Uint64{Uint64: uint64(v)}}
			case uint8:
				mv = QueryActionValue{Kind: &QueryActionValue_Uint64{Uint64: uint64(v)}}
			case uint16:
				mv = QueryActionValue{Kind: &QueryActionValue_Uint64{Uint64: uint64(v)}}
			case uint32:
				mv = QueryActionValue{Kind: &QueryActionValue_Uint64{Uint64: uint64(v)}}
			case uint64:
				mv = QueryActionValue{Kind: &QueryActionValue_Uint64{Uint64: v}}

			// TODO float32, float64, string, time.Time;

			// TODO (recursive) slices and maps

			default:
				panic(fmt.Sprintf("unhandled %[1]v (%[1]T)", v))
			}

			m.Map[column] = &mv
		}

		res.Res[i] = &m
	}

	return proto.Marshal(&res)
}

// UnmarshalActionQueryResult returns deserialized form of query Action result.
func UnmarshalActionQueryResult(b []byte) ([]map[string]interface{}, error) {
	var res QueryActionResult
	if err := proto.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(res.Res))

	for i, m := range res.Res {
		row := make(map[string]interface{}, len(m.Map))

		for mk, mv := range m.Map {
			var value interface{}

			switch v := mv.Kind.(type) {
			case *QueryActionValue_Nil:
				value = nil
			case *QueryActionValue_Bool:
				value = v.Bool
			case *QueryActionValue_Int64:
				value = v.Int64
			case *QueryActionValue_Uint64:
				value = v.Uint64
			default:
				panic(fmt.Sprintf("unhandled %[1]v (%[1]T)", v))
			}

			row[mk] = value
		}

		data[i] = row
	}

	return data, nil
}
