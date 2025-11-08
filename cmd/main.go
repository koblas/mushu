package main

import (
	"os"

	"github.com/koblas/mushu/internal/cli"
)

func main() {
	code := cli.Main()
	os.Exit(int(code))
}
