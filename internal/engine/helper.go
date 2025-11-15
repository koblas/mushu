package engine

import (
	"fmt"

	"go.starlark.net/starlark"
)

// convertValueToStarlark converts a Go value to a Starlark value
func convertValueToStarlark(value any) (starlark.Value, error) {
	if value == nil {
		return starlark.None, nil
	}

	switch v := value.(type) {
	case string:
		return starlark.String(v), nil

	case int:
		return starlark.MakeInt(v), nil
	case int8:
		return starlark.MakeInt(int(v)), nil
	case int16:
		return starlark.MakeInt(int(v)), nil
	case int32:
		return starlark.MakeInt(int(v)), nil
	case int64:
		return starlark.MakeInt64(v), nil

	case uint:
		return starlark.MakeUint(uint(v)), nil
	case uint8:
		return starlark.MakeUint(uint(v)), nil
	case uint16:
		return starlark.MakeUint(uint(v)), nil
	case uint32:
		return starlark.MakeUint(uint(v)), nil
	case uint64:
		return starlark.MakeUint64(v), nil

	case float32:
		return starlark.Float(v), nil
	case float64:
		return starlark.Float(v), nil

	case bool:
		return starlark.Bool(v), nil

	case []string:
		list := starlark.NewList(nil)
		for _, s := range v {
			if err := list.Append(starlark.String(s)); err != nil {
				return nil, fmt.Errorf("failed to append string to list: %w", err)
			}
		}
		return list, nil

	case []int:
		list := starlark.NewList(nil)
		for _, i := range v {
			if err := list.Append(starlark.MakeInt(i)); err != nil {
				return nil, fmt.Errorf("failed to append int to list: %w", err)
			}
		}
		return list, nil

	case []any:
		list := starlark.NewList(nil)
		for _, item := range v {
			converted, err := convertValueToStarlark(item)
			if err != nil {
				return nil, fmt.Errorf("failed to convert list item: %w", err)
			}
			if err := list.Append(converted); err != nil {
				return nil, fmt.Errorf("failed to append to list: %w", err)
			}
		}
		return list, nil

	case map[string]any:
		dict := starlark.NewDict(len(v))
		for key, val := range v {
			converted, err := convertValueToStarlark(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value for key %s: %w", key, err)
			}
			if err := dict.SetKey(starlark.String(key), converted); err != nil {
				return nil, fmt.Errorf("failed to set dict key %s: %w", key, err)
			}
		}
		return dict, nil

	case map[string]string:
		dict := starlark.NewDict(len(v))
		for key, val := range v {
			if err := dict.SetKey(starlark.String(key), starlark.String(val)); err != nil {
				return nil, fmt.Errorf("failed to set dict key %s: %w", key, err)
			}
		}
		return dict, nil

	case map[string]int:
		dict := starlark.NewDict(len(v))
		for key, val := range v {
			if err := dict.SetKey(starlark.String(key), starlark.MakeInt(val)); err != nil {
				return nil, fmt.Errorf("failed to set dict key %s: %w", key, err)
			}
		}
		return dict, nil

	case starlark.Value:
		// Already a Starlark value
		return v, nil

	default:
		return nil, fmt.Errorf("unsupported type: %T", value)
	}
}

// convertValueFromStarlark converts a Starlark value to a Go value
func convertValueFromStarlark(value starlark.Value) (any, error) {
	if value == nil || value == starlark.None {
		return nil, nil
	}

	switch v := value.(type) {
	case starlark.String:
		return string(v), nil

	case starlark.Int:
		if i64, ok := v.Int64(); ok {
			return int(i64), nil
		}
		if u64, ok := v.Uint64(); ok {
			return uint64(u64), nil
		}
		// For very large integers, return as string
		return v.String(), nil

	case starlark.Float:
		return float64(v), nil

	case starlark.Bool:
		return bool(v), nil

	case *starlark.List:
		result := make([]any, 0, v.Len())
		iter := v.Iterate()
		defer iter.Done()
		var val starlark.Value
		for iter.Next(&val) {
			converted, err := convertValueFromStarlark(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert list item: %w", err)
			}
			result = append(result, converted)
		}
		return result, nil

	case *starlark.Dict:
		result := make(map[string]any)
		for _, item := range v.Items() {
			if len(item) != 2 {
				return nil, fmt.Errorf("invalid dict item length: %d", len(item))
			}

			// Convert key (must be a string)
			keyStr, ok := item[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("dict key must be a string, got %T", item[0])
			}

			// Convert value
			val, err := convertValueFromStarlark(item[1])
			if err != nil {
				return nil, fmt.Errorf("failed to convert dict value for key %s: %w", keyStr, err)
			}

			result[string(keyStr)] = val
		}
		return result, nil

	case *starlark.Tuple:
		result := make([]any, 0, v.Len())
		iter := v.Iterate()
		defer iter.Done()
		var val starlark.Value
		for iter.Next(&val) {
			converted, err := convertValueFromStarlark(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert tuple item: %w", err)
			}
			result = append(result, converted)
		}
		return result, nil

	case *starlark.Set:
		result := make([]any, 0)
		iter := v.Iterate()
		defer iter.Done()
		var val starlark.Value
		for iter.Next(&val) {
			converted, err := convertValueFromStarlark(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert set item: %w", err)
			}
			result = append(result, converted)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported Starlark type: %T", value)
	}
}

// convertStringSliceToStarlark is a specialized helper for string slices
func convertStringSliceToStarlark(slice []string) *starlark.List {
	list := starlark.NewList(nil)
	for _, s := range slice {
		if err := list.Append(starlark.String(s)); err != nil {
			// Log error but continue - this shouldn't fail in normal operation
			continue
		}
	}
	return list
}

// convertStringSliceFromStarlark converts a Starlark list to a Go string slice
func convertStringSliceFromStarlark(list *starlark.List) ([]string, error) {
	result := make([]string, 0, list.Len())
	iter := list.Iterate()
	defer iter.Done()

	var val starlark.Value
	for iter.Next(&val) {
		str, ok := val.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", val)
		}
		result = append(result, string(str))
	}

	return result, nil
}

// convertDictToMap converts a Starlark dict to a Go map[string]any
func convertDictToMap(dict *starlark.Dict) (map[string]any, error) {
	result := make(map[string]any)

	for _, item := range dict.Items() {
		if len(item) != 2 {
			return nil, fmt.Errorf("invalid dict item length: %d", len(item))
		}

		// Convert key
		keyStr, ok := item[0].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("dict key must be a string, got %T", item[0])
		}

		// Convert value
		val, err := convertValueFromStarlark(item[1])
		if err != nil {
			return nil, fmt.Errorf("failed to convert dict value for key %s: %w", keyStr, err)
		}

		result[string(keyStr)] = val
	}

	return result, nil
}

// convertMapToDict converts a Go map to a Starlark dict
func convertMapToDict(m map[string]any) (*starlark.Dict, error) {
	dict := starlark.NewDict(len(m))

	for key, val := range m {
		converted, err := convertValueToStarlark(val)
		if err != nil {
			return nil, fmt.Errorf("failed to convert map value for key %s: %w", key, err)
		}

		if err := dict.SetKey(starlark.String(key), converted); err != nil {
			return nil, fmt.Errorf("failed to set dict key %s: %w", key, err)
		}
	}

	return dict, nil
}

// getStringFromDict safely retrieves a string value from a Starlark dict
func getStringFromDict(dict *starlark.Dict, key string) (string, bool) {
	val, ok, _ := dict.Get(starlark.String(key))
	if !ok {
		return "", false
	}

	str, ok := val.(starlark.String)
	if !ok {
		return "", false
	}

	return string(str), true
}

// getIntFromDict safely retrieves an int value from a Starlark dict
func getIntFromDict(dict *starlark.Dict, key string) (int, bool) {
	val, ok, _ := dict.Get(starlark.String(key))
	if !ok {
		return 0, false
	}

	intVal, ok := val.(starlark.Int)
	if !ok {
		return 0, false
	}

	i64, ok := intVal.Int64()
	if !ok {
		return 0, false
	}

	return int(i64), true
}

// getBoolFromDict safely retrieves a bool value from a Starlark dict
func getBoolFromDict(dict *starlark.Dict, key string) (bool, bool) {
	val, ok, _ := dict.Get(starlark.String(key))
	if !ok {
		return false, false
	}

	boolVal, ok := val.(starlark.Bool)
	if !ok {
		return false, false
	}

	return bool(boolVal), true
}
