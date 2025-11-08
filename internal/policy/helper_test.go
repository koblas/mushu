package policy

import (
	"testing"

	"go.starlark.net/starlark"
)

func TestConvertValueToStarlark_String(t *testing.T) {
	result, err := convertValueToStarlark("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str, ok := result.(starlark.String)
	if !ok {
		t.Fatalf("expected starlark.String, got %T", result)
	}

	if string(str) != "hello" {
		t.Errorf("expected 'hello', got '%s'", str)
	}
}

func TestConvertValueToStarlark_Int(t *testing.T) {
	result, err := convertValueToStarlark(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(starlark.Int)
	if !ok {
		t.Fatalf("expected starlark.Int, got %T", result)
	}

	i64, ok := intVal.Int64()
	if !ok {
		t.Fatalf("failed to convert to int64")
	}

	if i64 != 42 {
		t.Errorf("expected 42, got %d", i64)
	}
}

func TestConvertValueToStarlark_Bool(t *testing.T) {
	result, err := convertValueToStarlark(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal, ok := result.(starlark.Bool)
	if !ok {
		t.Fatalf("expected starlark.Bool, got %T", result)
	}

	if !bool(boolVal) {
		t.Errorf("expected true, got false")
	}
}

func TestConvertValueToStarlark_StringSlice(t *testing.T) {
	input := []string{"a", "b", "c"}
	result, err := convertValueToStarlark(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, ok := result.(*starlark.List)
	if !ok {
		t.Fatalf("expected *starlark.List, got %T", result)
	}

	if list.Len() != 3 {
		t.Errorf("expected length 3, got %d", list.Len())
	}

	// Check first element
	first, ok := list.Index(0).(starlark.String)
	if !ok || string(first) != "a" {
		t.Errorf("expected first element 'a', got %v", list.Index(0))
	}
}

func TestConvertValueToStarlark_Map(t *testing.T) {
	input := map[string]any{
		"name":  "test",
		"value": 123,
		"flag":  true,
	}

	result, err := convertValueToStarlark(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict, ok := result.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", result)
	}

	if dict.Len() != 3 {
		t.Errorf("expected length 3, got %d", dict.Len())
	}

	// Check name
	nameVal, ok, _ := dict.Get(starlark.String("name"))
	if !ok {
		t.Error("name key not found")
	}
	nameStr, ok := nameVal.(starlark.String)
	if !ok || string(nameStr) != "test" {
		t.Errorf("expected name='test', got %v", nameVal)
	}
}

func TestConvertValueFromStarlark_String(t *testing.T) {
	input := starlark.String("hello")
	result, err := convertValueFromStarlark(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}

	if str != "hello" {
		t.Errorf("expected 'hello', got '%s'", str)
	}
}

func TestConvertValueFromStarlark_Int(t *testing.T) {
	input := starlark.MakeInt(42)
	result, err := convertValueFromStarlark(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(int)
	if !ok {
		t.Fatalf("expected int, got %T", result)
	}

	if intVal != 42 {
		t.Errorf("expected 42, got %d", intVal)
	}
}

func TestConvertValueFromStarlark_Bool(t *testing.T) {
	input := starlark.Bool(true)
	result, err := convertValueFromStarlark(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", result)
	}

	if !boolVal {
		t.Errorf("expected true, got false")
	}
}

func TestConvertValueFromStarlark_List(t *testing.T) {
	list := starlark.NewList([]starlark.Value{
		starlark.String("a"),
		starlark.String("b"),
		starlark.String("c"),
	})

	result, err := convertValueFromStarlark(list)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slice, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(slice) != 3 {
		t.Errorf("expected length 3, got %d", len(slice))
	}

	if slice[0].(string) != "a" {
		t.Errorf("expected first element 'a', got %v", slice[0])
	}
}

func TestConvertValueFromStarlark_Dict(t *testing.T) {
	dict := starlark.NewDict(2)
	_ = dict.SetKey(starlark.String("name"), starlark.String("test"))
	_ = dict.SetKey(starlark.String("value"), starlark.MakeInt(123))

	result, err := convertValueFromStarlark(dict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if len(m) != 2 {
		t.Errorf("expected length 2, got %d", len(m))
	}

	if m["name"].(string) != "test" {
		t.Errorf("expected name='test', got %v", m["name"])
	}

	if m["value"].(int) != 123 {
		t.Errorf("expected value=123, got %v", m["value"])
	}
}

func TestConvertStringSliceToStarlark(t *testing.T) {
	input := []string{"x", "y", "z"}
	result := convertStringSliceToStarlark(input)

	if result.Len() != 3 {
		t.Errorf("expected length 3, got %d", result.Len())
	}

	first, ok := result.Index(0).(starlark.String)
	if !ok || string(first) != "x" {
		t.Errorf("expected first element 'x', got %v", result.Index(0))
	}
}

func TestConvertStringSliceFromStarlark(t *testing.T) {
	list := starlark.NewList([]starlark.Value{
		starlark.String("x"),
		starlark.String("y"),
		starlark.String("z"),
	})

	result, err := convertStringSliceFromStarlark(list)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected length 3, got %d", len(result))
	}

	if result[0] != "x" {
		t.Errorf("expected first element 'x', got '%s'", result[0])
	}
}

func TestGetStringFromDict(t *testing.T) {
	dict := starlark.NewDict(1)
	_ = dict.SetKey(starlark.String("key"), starlark.String("value"))

	result, ok := getStringFromDict(dict, "key")
	if !ok {
		t.Error("expected to find key")
	}

	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}

	// Test missing key
	_, ok = getStringFromDict(dict, "missing")
	if ok {
		t.Error("expected missing key to return false")
	}
}

func TestGetIntFromDict(t *testing.T) {
	dict := starlark.NewDict(1)
	_ = dict.SetKey(starlark.String("key"), starlark.MakeInt(42))

	result, ok := getIntFromDict(dict, "key")
	if !ok {
		t.Error("expected to find key")
	}

	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}

	// Test missing key
	_, ok = getIntFromDict(dict, "missing")
	if ok {
		t.Error("expected missing key to return false")
	}
}

func TestGetBoolFromDict(t *testing.T) {
	dict := starlark.NewDict(1)
	_ = dict.SetKey(starlark.String("key"), starlark.Bool(true))

	result, ok := getBoolFromDict(dict, "key")
	if !ok {
		t.Error("expected to find key")
	}

	if !result {
		t.Error("expected true, got false")
	}

	// Test missing key
	_, ok = getBoolFromDict(dict, "missing")
	if ok {
		t.Error("expected missing key to return false")
	}
}

func TestRoundTripConversion(t *testing.T) {
	// Test round-trip conversion: Go -> Starlark -> Go
	original := map[string]any{
		"name":   "test",
		"count":  42,
		"active": true,
		"tags":   []string{"a", "b", "c"},
		"nested": map[string]any{
			"key": "value",
		},
	}

	// Convert to Starlark
	starlarkVal, err := convertValueToStarlark(original)
	if err != nil {
		t.Fatalf("failed to convert to Starlark: %v", err)
	}

	// Convert back to Go
	result, err := convertValueFromStarlark(starlarkVal)
	if err != nil {
		t.Fatalf("failed to convert from Starlark: %v", err)
	}

	// Check the result
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if resultMap["name"].(string) != "test" {
		t.Errorf("name mismatch: expected 'test', got '%v'", resultMap["name"])
	}

	if resultMap["count"].(int) != 42 {
		t.Errorf("count mismatch: expected 42, got %v", resultMap["count"])
	}

	if resultMap["active"].(bool) != true {
		t.Errorf("active mismatch: expected true, got %v", resultMap["active"])
	}
}
