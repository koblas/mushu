package rules

// PRFile represents a file in a pull request
type PRFile struct {
	Filename  string
	Status    string // added, modified, removed, renamed
	Additions int
	Deletions int
	Changes   int
}
