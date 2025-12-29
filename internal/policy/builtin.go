package policy

import (
	"fmt"

	"github.com/koblas/mushu/internal/policy/re"
	"go.starlark.net/starlark"
)

func Stdlib() (starlark.StringDict, error) {
	result := starlark.StringDict{}

	union := func(a starlark.StringDict) {
		for _, key := range a.Keys() {
			result[key] = a[key]
		}
	}

	if value, err := re.LoadModule(); err != nil {
		return nil, fmt.Errorf("loading re module: %w", err)
	} else {
		union(value)
	}

	return result, nil
}
