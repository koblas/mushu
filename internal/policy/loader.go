package policy

import (
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/koblas/mushu/internal/policy/re"
	"go.starlark.net/starlark"
)

//go:embed builtin/*
var builtin embed.FS

var ErrNotFound = fmt.Errorf("builtin policy not found")

func Fetch(name string) (string, []byte, error) {
	data, err := builtin.ReadFile(path.Join("builtin", name+".star"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil, err
		}

		return "", nil, err
	}

	return name + ".star", data, nil
}

func Load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	if module == re.ModuleName {
		return re.LoadModule()
	}

	return starlark.StringDict{}, fmt.Errorf("module not found: %q", module)
}
