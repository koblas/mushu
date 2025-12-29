package starutil

import (
	"fmt"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

// Marshal turns go values into starlark types
func Marshal(data any) (starlark.Value, error) { //nolint:ireturn
	switch x := data.(type) {
	case starlark.Value:
		return x, nil
	// simple types
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(x), nil
	case string:
		return starlark.String(x), nil
	// integer types
	case int:
		return starlark.MakeInt(x), nil
	case int8:
		return starlark.MakeInt(int(x)), nil
	case int16:
		return starlark.MakeInt(int(x)), nil
	case int32:
		return starlark.MakeInt(int(x)), nil
	case int64:
		return starlark.MakeInt64(x), nil
	case uint:
		return starlark.MakeUint(x), nil
	case uint8:
		return starlark.MakeUint(uint(x)), nil
	case uint16:
		return starlark.MakeUint(uint(x)), nil
	case uint32:
		return starlark.MakeUint(uint(x)), nil
	case uint64:
		return starlark.MakeUint64(x), nil
	// float types
	case float32:
		return starlark.Float(float64(x)), nil
	case float64:
		return starlark.Float(x), nil
	case time.Time:
		return startime.Time(x), nil
	// the fun types
	case []string:
		elems := make([]starlark.Value, len(x))
		for i, val := range x {
			elems[i] = starlark.String(val)
		}

		return starlark.NewList(elems), nil
	case []int:
		elems := make([]starlark.Value, len(x))
		for i, val := range x {
			elems[i] = starlark.MakeInt(val)
		}

		return starlark.NewList(elems), nil
	case []any:
		elems := make([]starlark.Value, len(x))
		for i, val := range x {
			v, err := Marshal(val)
			if err != nil {
				return starlark.None, err
			}
			elems[i] = v
		}

		return starlark.NewList(elems), nil
	case map[any]any:
		dict := starlark.NewDict(len(x))
		for ki, val := range x {
			var key starlark.Value
			key, err := Marshal(ki)
			if err != nil {
				return starlark.None, fmt.Errorf("marshaling map key %v: %w", ki, err)
			}

			elem, err := Marshal(val)
			if err != nil {
				return starlark.None, fmt.Errorf("marshaling value %v: %w", val, err)
			}
			if err := dict.SetKey(key, elem); err != nil {
				return starlark.None, fmt.Errorf("setting dict key %v: %w", key, err)
			}
		}

		return dict, nil
	case map[string]string:
		dict := starlark.NewDict(len(x))
		for key, val := range x {
			elem := starlark.String(val)
			if err := dict.SetKey(starlark.String(key), elem); err != nil {
				return starlark.None, fmt.Errorf("setting dict key %s: %w", key, err)
			}
		}

		return dict, nil
	case map[string]any:
		dict := starlark.NewDict(len(x))
		for key, val := range x {
			elem, err := Marshal(val)
			if err != nil {
				return starlark.None, fmt.Errorf("marshaling map key %s: %w", key, err)
			}
			if err = dict.SetKey(starlark.String(key), elem); err != nil {
				return starlark.None, fmt.Errorf("setting dict key %s: %w", key, err)
			}
		}

		return dict, nil
	// case Marshaler:
	// v, err = x.MarshalStarlark()
	default:
		return starlark.None, fmt.Errorf("unrecognized type: %#v", x)
	}
}
