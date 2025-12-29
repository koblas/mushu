package policy

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"

	"github.com/koblas/mushu/internal/logging"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

//go:embed builtin/*
var builtin embed.FS

var ErrNotFound = errors.New("builtin policy not found")

func LoadBuiltin(ctx context.Context, name string) ([]byte, error) {
	data, err := builtin.ReadFile(path.Join("builtin", name+".star"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("builtin policy %q: %w", name, err)
		}

		return nil, fmt.Errorf("reading builtin policy %q: %w", name, err)
	}

	return data, nil
}

type loaderEntry struct {
	globals starlark.StringDict
	err     error
}

type PolicyLoader struct {
	cache       map[Ref]*loaderEntry
	fileOptions *syntax.FileOptions
	fs          fs.FS
	predeclared starlark.StringDict
}

func NewPolicyLoader(fs fs.FS, predeclared starlark.StringDict) *PolicyLoader {
	return &PolicyLoader{
		cache: make(map[Ref]*loaderEntry),
		fs:    fs,
		fileOptions: &syntax.FileOptions{
			Set:               false,
			While:             true,
			TopLevelControl:   false,
			GlobalReassign:    false,
			Recursion:         true,
			LoadBindsGlobally: false,
		},
		predeclared: predeclared,
	}
}

type loaderFunc func(thread *starlark.Thread, module string) (starlark.StringDict, error)

var ErrLoadCycle = errors.New("cycle detected in module loading")

// Additional builtin modules can be added here
func (l *PolicyLoader) Loader(ctx context.Context) loaderFunc {
	return func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		ref := RefParse(module)

		entry, exists := l.cache[ref]

		if entry == nil && exists {
			return nil, fmt.Errorf("module: %q: %w", module, ErrLoadCycle)
		}

		l.cache[ref] = nil // mark as loading

		data, err := l.SourceFetch(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("failed to load module %q: %w", module, err)
		}

		globals, err := starlark.ExecFileOptions(l.fileOptions, thread, ref.String(), data, l.predeclared)

		entry = &loaderEntry{
			globals: globals,
			err:     err,
		}
		l.cache[ref] = entry

		var evalErr *starlark.EvalError
		if err != nil && errors.As(err, &evalErr) {
			slog.ErrorContext(ctx, "starlark eval error",
				slog.String("module", module),
				slog.String("backtrace", evalErr.Backtrace()),
			)
		}

		return entry.globals, entry.err
	}
}

func (l *PolicyLoader) SourceFetch(ctx context.Context, ref Ref) ([]byte, error) {
	fd, err := l.fs.Open(path.Join("policies", ref.Name+".star"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.DebugContext(ctx, "policy not found attempting builtin",
				slog.String("name", ref.Name),
				logging.Err(err),
			)

			if data, err := LoadBuiltin(ctx, ref.Name); err == nil {
				// If there was an error, returning it from caller
				return data, nil
			}
		}

		slog.InfoContext(ctx, "Opening policy from FS failed",
			slog.String("name", ref.Name),
			logging.Err(err),
		)

		return nil, fmt.Errorf("opening policy file %s: %w", ref.Name, err)
	}

	defer func() {
		_ = fd.Close()
	}()

	b, err := io.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	return b, nil
}
