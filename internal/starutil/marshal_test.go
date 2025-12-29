package starutil_test

import (
	"testing"
	"time"

	"github.com/koblas/mushu/internal/starutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

func TestMarshal_SimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected starlark.Value
	}{
		{
			name:     "nil",
			input:    nil,
			expected: starlark.None,
		},
		{
			name:     "bool true",
			input:    true,
			expected: starlark.Bool(true),
		},
		{
			name:     "bool false",
			input:    false,
			expected: starlark.Bool(false),
		},
		{
			name:     "string",
			input:    "hello",
			expected: starlark.String("hello"),
		},
		{
			name:     "empty string",
			input:    "",
			expected: starlark.String(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := starutil.Marshal(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMarshal_IntegerTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"int", int(42), 42},
		{"int8", int8(42), 42},
		{"int16", int16(42), 42},
		{"int32", int32(42), 42},
		{"int64", int64(42), 42},
		{"uint", uint(42), 42},
		{"uint8", uint8(42), 42},
		{"uint16", uint16(42), 42},
		{"uint32", uint32(42), 42},
		{"uint64", uint64(42), 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := starutil.Marshal(tt.input)
			require.NoError(t, err)

			// Convert back to int to compare
			var val int
			err = starlark.AsInt(result, &val)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func TestMarshal_FloatTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"float32", float32(3.14), 3.14},
		{"float64", float64(3.14159), 3.14159},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := starutil.Marshal(tt.input)
			require.NoError(t, err)

			val, ok := starlark.AsFloat(result)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, val, 0.00001)
		})
	}
}

func TestMarshal_Time(t *testing.T) {
	now := time.Now()
	result, err := starutil.Marshal(now)
	require.NoError(t, err)

	starTime, ok := result.(startime.Time)
	require.True(t, ok)
	assert.Equal(t, now, time.Time(starTime))
}

func TestMarshal_StringSlice(t *testing.T) {
	input := []string{"a", "b", "c"}
	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	list, ok := result.(*starlark.List)
	require.True(t, ok)
	assert.Equal(t, 3, list.Len())

	assert.Equal(t, starlark.String("a"), list.Index(0))
	assert.Equal(t, starlark.String("b"), list.Index(1))
	assert.Equal(t, starlark.String("c"), list.Index(2))
}

func TestMarshal_IntSlice(t *testing.T) {
	input := []int{1, 2, 3}
	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	list, ok := result.(*starlark.List)
	require.True(t, ok)
	assert.Equal(t, 3, list.Len())

	for i := range 3 {
		var val int
		err := starlark.AsInt(list.Index(i), &val)
		require.NoError(t, err)
		assert.Equal(t, i+1, val)
	}
}

func TestMarshal_AnySlice(t *testing.T) {
	input := []any{"string", 42, true, 3.14}
	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	list, ok := result.(*starlark.List)
	require.True(t, ok)
	assert.Equal(t, 4, list.Len())

	assert.Equal(t, starlark.String("string"), list.Index(0))

	var intVal int
	err = starlark.AsInt(list.Index(1), &intVal)
	require.NoError(t, err)
	assert.Equal(t, 42, intVal)

	assert.Equal(t, starlark.Bool(true), list.Index(2))

	floatVal, ok := starlark.AsFloat(list.Index(3))
	require.True(t, ok)
	assert.InDelta(t, 3.14, floatVal, 0.001)
}

func TestMarshal_MapStringString(t *testing.T) {
	input := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	dict, ok := result.(*starlark.Dict)
	require.True(t, ok)
	assert.Equal(t, 2, dict.Len())

	val1, found, err := dict.Get(starlark.String("key1"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, starlark.String("value1"), val1)

	val2, found, err := dict.Get(starlark.String("key2"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, starlark.String("value2"), val2)
}

func TestMarshal_MapStringAny(t *testing.T) {
	input := map[string]any{
		"name":   "Alice",
		"age":    30,
		"active": true,
	}
	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	dict, ok := result.(*starlark.Dict)
	require.True(t, ok)
	assert.Equal(t, 3, dict.Len())

	name, found, err := dict.Get(starlark.String("name"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, starlark.String("Alice"), name)

	age, found, err := dict.Get(starlark.String("age"))
	require.NoError(t, err)
	require.True(t, found)
	var ageVal int
	err = starlark.AsInt(age, &ageVal)
	require.NoError(t, err)
	assert.Equal(t, 30, ageVal)

	active, found, err := dict.Get(starlark.String("active"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, starlark.Bool(true), active)
}

func TestMarshal_MapAnyAny(t *testing.T) {
	input := map[any]any{
		"string_key": "value",
		42:           "int_key",
		true:         false,
	}
	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	dict, ok := result.(*starlark.Dict)
	require.True(t, ok)
	assert.Equal(t, 3, dict.Len())

	val1, found, err := dict.Get(starlark.String("string_key"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, starlark.String("value"), val1)
}

func TestMarshal_NestedStructures(t *testing.T) {
	input := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
		},
		"count": 2,
	}

	result, err := starutil.Marshal(input)
	require.NoError(t, err)

	dict, ok := result.(*starlark.Dict)
	require.True(t, ok)

	users, found, err := dict.Get(starlark.String("users"))
	require.NoError(t, err)
	require.True(t, found)

	usersList, ok := users.(*starlark.List)
	require.True(t, ok)
	assert.Equal(t, 2, usersList.Len())
}

func TestMarshal_StarlarkValue(t *testing.T) {
	// Passing a starlark.Value should return it unchanged
	input := starlark.String("already starlark")
	result, err := starutil.Marshal(input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestMarshal_UnsupportedType(t *testing.T) {
	type customStruct struct {
		Field string
	}

	input := customStruct{Field: "value"}
	_, err := starutil.Marshal(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized type")
}

func TestMarshal_EmptyCollections(t *testing.T) {
	t.Run("empty string slice", func(t *testing.T) {
		result, err := starutil.Marshal([]string{})
		require.NoError(t, err)
		list, ok := result.(*starlark.List)
		require.True(t, ok)
		assert.Equal(t, 0, list.Len())
	})

	t.Run("empty int slice", func(t *testing.T) {
		result, err := starutil.Marshal([]int{})
		require.NoError(t, err)
		list, ok := result.(*starlark.List)
		require.True(t, ok)
		assert.Equal(t, 0, list.Len())
	})

	t.Run("empty any slice", func(t *testing.T) {
		result, err := starutil.Marshal([]any{})
		require.NoError(t, err)
		list, ok := result.(*starlark.List)
		require.True(t, ok)
		assert.Equal(t, 0, list.Len())
	})

	t.Run("empty map", func(t *testing.T) {
		result, err := starutil.Marshal(map[string]string{})
		require.NoError(t, err)
		dict, ok := result.(*starlark.Dict)
		require.True(t, ok)
		assert.Equal(t, 0, dict.Len())
	})
}

func TestMarshal_ErrorCases(t *testing.T) {
	t.Run("any slice with unsupported type", func(t *testing.T) {
		type customType struct {
			Field string
		}
		input := []any{"valid", customType{"invalid"}}
		_, err := starutil.Marshal(input)
		require.Error(t, err)
	})

	t.Run("map[any]any with unmarshalable key", func(t *testing.T) {
		type customType struct {
			Field string
		}
		input := map[any]any{
			customType{"bad"}: "value",
		}
		_, err := starutil.Marshal(input)
		require.Error(t, err)
	})

	t.Run("map[any]any with unmarshalable value", func(t *testing.T) {
		type customType struct {
			Field string
		}
		input := map[any]any{
			"key": customType{"bad"},
		}
		_, err := starutil.Marshal(input)
		require.Error(t, err)
	})

	t.Run("map[string]any with unmarshalable value", func(t *testing.T) {
		type customType struct {
			Field string
		}
		input := map[string]any{
			"key": customType{"bad"},
		}
		_, err := starutil.Marshal(input)
		require.Error(t, err)
	})
}
