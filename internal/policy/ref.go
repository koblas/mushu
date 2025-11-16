package policy

import "github.com/koblas/mushu/internal/policy/re"

type Source int

const (
	SourceBuiltin Source = iota
	SourceFile    Source = iota
)

func (s Source) String() string {
	switch s {
	case SourceBuiltin:
		return "builtin"
	case SourceFile:
		return "file"
	default:
		return "unknown"
	}
}

type Ref struct {
	Source Source
	Name   string
}

func RefParse(name string) Ref {
	source := SourceFile
	if name == re.ModuleName {
		source = SourceBuiltin
	}

	return Ref{
		Source: source,
		Name:   name,
	}
}

func (r *Ref) String() string {
	return r.Source.String() + ":///" + r.Name
}
