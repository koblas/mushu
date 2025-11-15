package starutil_test

import (
	"testing"
	"time"

	"github.com/koblas/mushu/internal/starutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTrip tests that Marshal followed by Unmarshal returns the original value
// Note: Slices and maps will be returned as []interface{} and map[string]interface{}
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any // if different from input
	}{
		{"nil", nil, nil},
		{"bool true", true, nil},
		{"bool false", false, nil},
		{"string", "hello world", nil},
		{"empty string", "", nil},
		{"int", 42, nil},
		{"negative int", -42, nil},
		{"zero", 0, nil},
		{"float", 3.14159, nil},
		{"string slice", []string{"a", "b", "c"}, []interface{}{"a", "b", "c"}},
		{"empty string slice", []string{}, []interface{}{}},
		{"int slice", []int{1, 2, 3}, []interface{}{1, 2, 3}},
		{"empty int slice", []int{}, []interface{}{}},
		{"map string to string", map[string]string{"key": "value"}, map[string]any{"key": "value"}},
		{"empty map", map[string]string{}, map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to starlark
			starlarkVal, err := starutil.Marshal(tt.input)
			require.NoError(t, err)

			// Unmarshal back to go
			result, err := starutil.Unmarshal(starlarkVal)
			require.NoError(t, err)

			// Compare
			expected := tt.expected
			if expected == nil {
				expected = tt.input
			}
			assert.Equal(t, expected, result)
		})
	}
}

func TestRoundTrip_ComplexStructures(t *testing.T) {
	t.Run("map with mixed values", func(t *testing.T) {
		input := map[string]any{
			"name":   "Alice",
			"age":    30,
			"active": true,
			"score":  95.5,
		}

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Alice", resultMap["name"])
		assert.Equal(t, 30, resultMap["age"])
		assert.Equal(t, true, resultMap["active"])
		assert.InDelta(t, 95.5, resultMap["score"].(float64), 0.001)
	})

	t.Run("nested maps", func(t *testing.T) {
		input := map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"address": map[string]any{
					"city": "NYC",
					"zip":  "10001",
				},
			},
		}

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)

		user, ok := resultMap["user"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Alice", user["name"])

		address, ok := user["address"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "NYC", address["city"])
		assert.Equal(t, "10001", address["zip"])
	})

	t.Run("slice of maps", func(t *testing.T) {
		input := []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
		}

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		resultSlice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 2, len(resultSlice))

		first, ok := resultSlice[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Alice", first["name"])
		assert.Equal(t, 30, first["age"])
	})

	t.Run("map with slice values", func(t *testing.T) {
		input := map[string]any{
			"numbers": []int{1, 2, 3},
			"words":   []string{"a", "b", "c"},
		}

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)

		numbers, ok := resultMap["numbers"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(numbers))
		assert.Equal(t, 1, numbers[0])
		assert.Equal(t, 2, numbers[1])
		assert.Equal(t, 3, numbers[2])

		words, ok := resultMap["words"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(words))
		assert.Equal(t, "a", words[0])
		assert.Equal(t, "b", words[1])
		assert.Equal(t, "c", words[2])
	})
}

func TestRoundTrip_Time(t *testing.T) {
	// Time values should round-trip successfully
	now := time.Now().UTC().Truncate(time.Second) // Truncate to avoid precision issues

	starlarkVal, err := starutil.Marshal(now)
	require.NoError(t, err)

	result, err := starutil.Unmarshal(starlarkVal)
	require.NoError(t, err)

	resultTime, ok := result.(time.Time)
	require.True(t, ok)
	assert.True(t, now.Equal(resultTime))
}

func TestRoundTrip_SpecialCases(t *testing.T) {
	t.Run("nil in any slice", func(t *testing.T) {
		input := []any{"value", nil, "another"}

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		resultSlice, ok := result.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(resultSlice))
		assert.Equal(t, "value", resultSlice[0])
		assert.Nil(t, resultSlice[1])
		assert.Equal(t, "another", resultSlice[2])
	})

	t.Run("unicode strings", func(t *testing.T) {
		input := "Hello 世界 🌍"

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		assert.Equal(t, input, result)
	})

	t.Run("large numbers", func(t *testing.T) {
		input := 1234567890

		starlarkVal, err := starutil.Marshal(input)
		require.NoError(t, err)

		result, err := starutil.Unmarshal(starlarkVal)
		require.NoError(t, err)

		assert.Equal(t, input, result)
	})
}
