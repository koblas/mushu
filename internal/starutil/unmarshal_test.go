package starutil_test

import (
	"testing"
	"time"

	"github.com/koblas/mushu/internal/starutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func TestUnmarshal_SimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    starlark.Value
		expected any
	}{
		{
			name:     "none",
			input:    starlark.None,
			expected: nil,
		},
		{
			name:     "bool true",
			input:    starlark.Bool(true),
			expected: true,
		},
		{
			name:     "bool false",
			input:    starlark.Bool(false),
			expected: false,
		},
		{
			name:     "string",
			input:    starlark.String("hello"),
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    starlark.String(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := starutil.Unmarshal(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmarshal_Int(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"zero", 0, 0},
		{"positive", 42, 42},
		{"negative", -42, -42},
		{"large", 1000000, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := starlark.MakeInt(tt.value)
			result, err := starutil.Unmarshal(input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmarshal_Float(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{"zero", 0.0, 0.0},
		{"positive", 3.14, 3.14},
		{"negative", -3.14, -3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := starlark.Float(tt.value)
			result, err := starutil.Unmarshal(input)
			require.NoError(t, err)
			floatResult, ok := result.(float64)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatResult, 0.00001)
		})
	}
}

func TestUnmarshal_Time(t *testing.T) {
	now := time.Now()
	input := startime.Time(now)

	result, err := starutil.Unmarshal(input)
	require.NoError(t, err)

	timeResult, ok := result.(time.Time)
	require.True(t, ok)
	assert.Equal(t, now, timeResult)
}

func TestUnmarshal_List(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		input := starlark.NewList([]starlark.Value{})
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		slice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 0, len(slice))
	})

	t.Run("list of strings", func(t *testing.T) {
		input := starlark.NewList([]starlark.Value{
			starlark.String("a"),
			starlark.String("b"),
			starlark.String("c"),
		})
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		slice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(slice))
		assert.Equal(t, "a", slice[0])
		assert.Equal(t, "b", slice[1])
		assert.Equal(t, "c", slice[2])
	})

	t.Run("list of ints", func(t *testing.T) {
		input := starlark.NewList([]starlark.Value{
			starlark.MakeInt(1),
			starlark.MakeInt(2),
			starlark.MakeInt(3),
		})
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		slice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(slice))
		assert.Equal(t, 1, slice[0])
		assert.Equal(t, 2, slice[1])
		assert.Equal(t, 3, slice[2])
	})

	t.Run("list of mixed types", func(t *testing.T) {
		input := starlark.NewList([]starlark.Value{
			starlark.String("string"),
			starlark.MakeInt(42),
			starlark.Bool(true),
			starlark.Float(3.14),
		})
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		slice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 4, len(slice))
		assert.Equal(t, "string", slice[0])
		assert.Equal(t, 42, slice[1])
		assert.Equal(t, true, slice[2])
		assert.InDelta(t, 3.14, slice[3].(float64), 0.001)
	})
}

func TestUnmarshal_Tuple(t *testing.T) {
	t.Run("empty tuple", func(t *testing.T) {
		input := starlark.Tuple([]starlark.Value{})
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		slice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 0, len(slice))
	})

	t.Run("tuple of values", func(t *testing.T) {
		input := starlark.Tuple([]starlark.Value{
			starlark.String("a"),
			starlark.MakeInt(1),
			starlark.Bool(true),
		})
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		slice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(slice))
		assert.Equal(t, "a", slice[0])
		assert.Equal(t, 1, slice[1])
		assert.Equal(t, true, slice[2])
	})
}

func TestUnmarshal_Dict_StringKeys(t *testing.T) {
	t.Run("empty dict", func(t *testing.T) {
		input := starlark.NewDict(0)
		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 0, len(dict))
	})

	t.Run("string keys", func(t *testing.T) {
		input := starlark.NewDict(2)
		err := input.SetKey(starlark.String("name"), starlark.String("Alice"))
		require.NoError(t, err)
		err = input.SetKey(starlark.String("age"), starlark.MakeInt(30))
		require.NoError(t, err)

		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(dict))
		assert.Equal(t, "Alice", dict["name"])
		assert.Equal(t, 30, dict["age"])
	})

	t.Run("nested dict with string keys", func(t *testing.T) {
		nested := starlark.NewDict(1)
		err := nested.SetKey(starlark.String("city"), starlark.String("NYC"))
		require.NoError(t, err)

		input := starlark.NewDict(2)
		err = input.SetKey(starlark.String("name"), starlark.String("Alice"))
		require.NoError(t, err)
		err = input.SetKey(starlark.String("address"), nested)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(input)
		require.NoError(t, err)

		dict, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Alice", dict["name"])

		address, ok := dict["address"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "NYC", address["city"])
	})
}

func TestUnmarshal_Dict_MixedKeys(t *testing.T) {
	input := starlark.NewDict(3)
	err := input.SetKey(starlark.String("string_key"), starlark.String("value1"))
	require.NoError(t, err)
	err = input.SetKey(starlark.MakeInt(42), starlark.String("value2"))
	require.NoError(t, err)
	err = input.SetKey(starlark.Bool(true), starlark.String("value3"))
	require.NoError(t, err)

	result, err := starutil.Unmarshal(input)
	require.NoError(t, err)

	dict, ok := result.(map[any]any)
	require.True(t, ok)
	assert.Equal(t, 3, len(dict))
	assert.Equal(t, "value1", dict["string_key"])
	assert.Equal(t, "value2", dict[42])
	assert.Equal(t, "value3", dict[true])
}

func TestUnmarshal_NestedStructures(t *testing.T) {
	// Create a nested structure: {users: [{name: "Alice", age: 30}, {name: "Bob", age: 25}]}
	user1 := starlark.NewDict(2)
	err := user1.SetKey(starlark.String("name"), starlark.String("Alice"))
	require.NoError(t, err)
	err = user1.SetKey(starlark.String("age"), starlark.MakeInt(30))
	require.NoError(t, err)

	user2 := starlark.NewDict(2)
	err = user2.SetKey(starlark.String("name"), starlark.String("Bob"))
	require.NoError(t, err)
	err = user2.SetKey(starlark.String("age"), starlark.MakeInt(25))
	require.NoError(t, err)

	usersList := starlark.NewList([]starlark.Value{user1, user2})

	input := starlark.NewDict(1)
	err = input.SetKey(starlark.String("users"), usersList)
	require.NoError(t, err)

	result, err := starutil.Unmarshal(input)
	require.NoError(t, err)

	dict, ok := result.(map[string]any)
	require.True(t, ok)

	users, ok := dict["users"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(users))

	firstUser, ok := users[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Alice", firstUser["name"])
	assert.Equal(t, 30, firstUser["age"])

	secondUser, ok := users[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Bob", secondUser["name"])
	assert.Equal(t, 25, secondUser["age"])
}

func TestUnmarshal_Set(t *testing.T) {
	input := starlark.NewSet(1)
	err := input.Insert(starlark.String("value"))
	require.NoError(t, err)

	_, err = starutil.Unmarshal(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sets aren't yet supported")
}

func TestUnmarshal_Struct(t *testing.T) {
	// Create a simple starlark struct
	input := starlarkstruct.FromStringDict(starlark.String("TestStruct"), starlark.StringDict{
		"field": starlark.String("value"),
	})

	_, err := starutil.Unmarshal(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestUnmarshal_UnsupportedType(t *testing.T) {
	// Create a custom starlark type (using a function as an example)
	input := starlark.NewBuiltin("test", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.None, nil
	})

	_, err := starutil.Unmarshal(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized starlark type")
}
