package starutil

import (
	"errors"
	"fmt"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Unmarshal decodes a starlark.Value into it's golang counterpart
func Unmarshal(x starlark.Value) (any, error) {
	switch v := x.(type) {
	case starlark.NoneType:
		return nil, nil //nolint:nilnil
	case starlark.Bool:
		return v.Truth() == starlark.True, nil
	case starlark.Int:
		var tmp int
		err := starlark.AsInt(x, &tmp)
		if err != nil {
			return 0, fmt.Errorf("unmarshaling starlark int: %w", err)
		}

		return tmp, nil
	case starlark.Float:
		f, ok := starlark.AsFloat(x)
		if !ok {
			return nil, errors.New("couldn't parse float")
		}

		return f, nil
	case starlark.String:
		return v.GoString(), nil
	case startime.Time:
		return time.Time(v), nil
	case *starlark.Dict:
		allStrings := true
		for _, key := range v.Keys() {
			allStrings = allStrings && key.Type() == "string"
		}

		if allStrings {
			result := make(map[string]any, v.Len())
			for key, val := range starlark.Entries(v) {
				strKey, ok := key.(starlark.String)
				if !ok {
					return nil, errors.New("unmarshaling starlark dict key to string")
				}

				dictVal, err := Unmarshal(val)
				if err != nil {
					return nil, fmt.Errorf("unmarshaling starlark dict value: %w", err)
				}

				result[strKey.GoString()] = dictVal
			}

			return result, nil
		}

		result := make(map[any]any, v.Len())
		for key, val := range starlark.Entries(v) {
			dictKey, err := Unmarshal(key)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling starlark dict key: %w", err)
			}

			dictVal, err := Unmarshal(val)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling starlark dict value: %w", err)
			}

			result[dictKey] = dictVal
		}

		return result, nil
	case *starlark.List:
		value := make([]any, 0, v.Len())
		for val := range starlark.Elements(v) {
			val, err := Unmarshal(val)
			if err != nil {
				return nil, err
			}
			value = append(value, val)
		}

		return value, nil
	case starlark.Tuple:
		value := make([]any, 0, v.Len())
		for val := range starlark.Elements(v) {
			val, err := Unmarshal(val)
			if err != nil {
				return nil, err
			}
			value = append(value, val)
		}

		return value, nil
	case *starlark.Set:
		return nil, errors.New("sets aren't yet supported")
	case *starlarkstruct.Struct:
		return nil, fmt.Errorf("constructor object from *starlarkstruct.Struct not supported Marshaler to starlark object: %s", v.Constructor().Type()) //nolint:lll
	default:
		return nil, fmt.Errorf("unrecognized starlark type: %s", x.Type())
	}
}
