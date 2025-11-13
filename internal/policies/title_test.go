package main

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/koblas/mushu/internal/policy"
	"github.com/stretchr/testify/assert"
)

//go:embed title.star
var titlePolicy string

func TestHelloWorld(t *testing.T) {
	engine := policy.NewPolicyEngine(nil, titlePolicy)

	fmt.Println(titlePolicy)

	prData := policy.PRData{
		Title: "WIP: Add new feature",
	}

	res, err := engine.EvaluatePR(t.Context(), &prData)

	assert.NoError(t, err)

	assert.Equal(t, "deny", res.Decision)
}
