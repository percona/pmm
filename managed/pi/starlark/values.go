// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package starlark

import (
	"fmt"
	"time"

	"go.starlark.net/starlark"
)

// goToStarlark converts Go value to Starlark value.
// Supported types:
//   - nil -> NoneType (None);
//   - bool -> bool;
//   - int64, uint64 -> int;
//   - float64 -> float;
//   - string, []byte -> string;
//   - time.Time -> int (UNIX timestamp in nanoseconds);
//   - []interface{} -> list;
//   - map[string]interface{} -> dict.
func goToStarlark(v any) (starlark.Value, error) { //nolint: ireturn
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

	case []any:
		res := make([]starlark.Value, len(v))
		for i, el := range v {
			sv, err := goToStarlark(el)
			if err != nil {
				return nil, err
			}

			res[i] = sv
		}

		return starlark.NewList(res), nil

	case []map[string]any:
		res := make([]starlark.Value, len(v))
		for i, el := range v {
			sv, err := goToStarlark(el)
			if err != nil {
				return nil, err
			}

			res[i] = sv
		}

		return starlark.NewList(res), nil

	case [][]map[string]any:
		res := make([]starlark.Value, len(v))
		for i, el := range v {
			sv, err := goToStarlark(el)
			if err != nil {
				return nil, err
			}

			res[i] = sv
		}

		return starlark.NewList(res), nil

	case map[string]any:
		res := starlark.NewDict(len(v))
		for k, gv := range v {
			sv, err := goToStarlark(gv)
			if err != nil {
				return nil, err
			}

			err = res.SetKey(starlark.String(k), sv)
			if err != nil {
				return nil, err
			}
		}

		return res, nil

	default:
		return nil, fmt.Errorf("unhandled type %[1]T (%[1]v)", v)
	}
}

// starlarkToGo converts Starlark value to Go value.
// Supported types:
//   - NoneType -> nil;
//   - bool -> bool;
//   - int -> int64 or uint64;
//   - float -> float64;
//   - string -> string;
//   - tuple -> []interface{}
//   - list -> []interface{}
//   - dict (with string keys) -> map[string]interface{}.
func starlarkToGo(v starlark.Value) (any, error) {
	switch v := v.(type) {
	case starlark.NoneType:
		return nil, nil //nolint:nilnil //intended

	case starlark.Bool:
		return bool(v), nil

	case starlark.Int:
		if i, ok := v.Int64(); ok {
			return i, nil
		}

		if u, ok := v.Uint64(); ok {
			return u, nil
		}

		return nil, fmt.Errorf("integer value %s is too big", v)

	case starlark.Float:
		return float64(v), nil

	case starlark.String:
		return string(v), nil

	case starlark.Tuple:
		res := make([]any, len(v))
		for i, el := range v {
			gv, err := starlarkToGo(el)
			if err != nil {
				return nil, err
			}

			res[i] = gv
		}

		return res, nil

	case *starlark.List:
		res := make([]any, v.Len())
		for i := range v.Len() {
			gv, err := starlarkToGo(v.Index(i))
			if err != nil {
				return nil, err
			}

			res[i] = gv
		}

		return res, nil

	case *starlark.Dict:
		res := make(map[string]any, v.Len())
		for _, tu := range v.Items() {
			k, v := tu[0], tu[1]
			ks, ok := k.(starlark.String)

			if !ok {
				return nil, fmt.Errorf("unhandled dict key type %[1]T (%[1]v)", k)
			}

			gv, err := starlarkToGo(v)
			if err != nil {
				return nil, err
			}

			res[string(ks)] = gv
		}

		return res, nil

	default:
		return nil, fmt.Errorf("unhandled type %T", v)
	}
}
