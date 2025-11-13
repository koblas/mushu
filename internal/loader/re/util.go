package re

import (
	"fmt"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

// Marshal turns go values into starlark types
func Marshal(data interface{}) (v starlark.Value, err error) {
	switch x := data.(type) {
	case nil:
		v = starlark.None
	case bool:
		v = starlark.Bool(x)
	case string:
		v = starlark.String(x)
	case int:
		v = starlark.MakeInt(x)
	case int8:
		v = starlark.MakeInt(int(x))
	case int16:
		v = starlark.MakeInt(int(x))
	case int32:
		v = starlark.MakeInt(int(x))
	case int64:
		v = starlark.MakeInt64(x)
	case uint:
		v = starlark.MakeUint(x)
	case uint8:
		v = starlark.MakeUint(uint(x))
	case uint16:
		v = starlark.MakeUint(uint(x))
	case uint32:
		v = starlark.MakeUint(uint(x))
	case uint64:
		v = starlark.MakeUint64(x)
	case float32:
		v = starlark.Float(float64(x))
	case float64:
		v = starlark.Float(x)
	case time.Time:
		v = startime.Time(x)
	case []interface{}:
		var elems = make([]starlark.Value, len(x))
		for i, val := range x {
			elems[i], err = Marshal(val)
			if err != nil {
				return
			}
		}
		v = starlark.NewList(elems)
	case map[interface{}]interface{}:
		dict := &starlark.Dict{}
		var elem starlark.Value
		for ki, val := range x {
			var key starlark.Value
			key, err = Marshal(ki)
			if err != nil {
				return
			}

			elem, err = Marshal(val)
			if err != nil {
				return
			}
			if err = dict.SetKey(key, elem); err != nil {
				return
			}
		}
		v = dict
	case map[string]interface{}:
		dict := &starlark.Dict{}
		var elem starlark.Value
		for key, val := range x {
			elem, err = Marshal(val)
			if err != nil {
				return
			}
			if err = dict.SetKey(starlark.String(key), elem); err != nil {
				return
			}
		}
		v = dict
	// case Marshaler:
	// v, err = x.MarshalStarlark()
	default:
		return starlark.None, fmt.Errorf("unrecognized type: %#v", x)
	}
	return
}
