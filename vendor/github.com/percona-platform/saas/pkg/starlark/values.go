package starlark

import (
	"time"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

// goToStarlark converts Go value to Starlark value.
// Supported types:
//  * nil -> NoneType (None);
//  * bool -> bool;
//  * int64, uint64 -> int;
//  * float64 -> float;
//  * string, []byte -> string;
//  * time.Time -> int (UNIX timestamp in nanoseconds);
//  * []interface{} -> list;
//  * map[string]interface{} -> dict.
func goToStarlark(v interface{}) (starlark.Value, error) {
	switch v := v.(type) {
	case nil:
		return starlark.None, nil

	case bool:
		return starlark.Bool(v), nil

	case int64:
		return starlark.MakeInt64(v), nil

	case uint64:
		return starlark.MakeUint64(v), nil

	case float64:
		return starlark.Float(v), nil

	case []byte:
		return starlark.String(v), nil

	case string:
		return starlark.String(v), nil

	case time.Time:
		return starlark.MakeInt64(v.UnixNano()), nil

	case []interface{}:
		res := make([]starlark.Value, len(v))
		for i, el := range v {
			sv, err := goToStarlark(el)
			if err != nil {
				return nil, err
			}
			res[i] = sv
		}
		return starlark.NewList(res), nil

	case map[string]interface{}:
		res := starlark.NewDict(len(v))
		for k, gv := range v {
			sv, err := goToStarlark(gv)
			if err != nil {
				return nil, err
			}
			if err := res.SetKey(starlark.String(k), sv); err != nil {
				return nil, errors.Wrapf(err, "failed to add %[1]v (%[1]T) = %[2]v (%[2]T) to dict", k, gv)
			}
		}
		return res, nil

	default:
		return nil, errors.Errorf("unhandled type %[1]T (%[1]v)", v)
	}
}

// starlarkToGo converts Starlark value to Go value.
// Supported types:
//  * NoneType -> nil;
//  * bool -> bool;
//  * int -> int64 or uint64;
//  * float -> float64;
//  * string -> string;
//  * tuple -> []interface{}
//  * list -> []interface{}
//  * dict (with string keys) -> map[string]interface{}.
func starlarkToGo(v starlark.Value) (interface{}, error) { //nolint:funlen
	switch v := v.(type) {
	case starlark.NoneType:
		return nil, nil

	case starlark.Bool:
		return bool(v), nil

	case starlark.Int:
		if i, ok := v.Int64(); ok {
			return i, nil
		}
		if u, ok := v.Uint64(); ok {
			return u, nil
		}
		return nil, errors.Errorf("interger value %s is too big", v)

	case starlark.Float:
		return float64(v), nil

	case starlark.String:
		return string(v), nil

	case starlark.Tuple:
		res := make([]interface{}, len(v))
		for i, el := range v {
			gv, err := starlarkToGo(el)
			if err != nil {
				return nil, err
			}
			res[i] = gv
		}
		return res, nil

	case *starlark.List:
		res := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			gv, err := starlarkToGo(v.Index(i))
			if err != nil {
				return nil, err
			}
			res[i] = gv
		}
		return res, nil

	case *starlark.Dict:
		res := make(map[string]interface{}, v.Len())
		for _, tu := range v.Items() {
			k, v := tu[0], tu[1]
			ks, ok := k.(starlark.String)
			if !ok {
				return nil, errors.Errorf("unhandled dict key type %[1]T (%[1]v)", k)
			}
			gv, err := starlarkToGo(v)
			if err != nil {
				return nil, err
			}
			res[string(ks)] = gv
		}
		return res, nil

	default:
		return nil, errors.Errorf("unhandled type %T", v)
	}
}
