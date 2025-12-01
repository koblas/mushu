package action_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/koblas/mushu/internal/action"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const content = `package toolkit

// placeholder for go docs
`

func TestClient(t *testing.T) {
	repo, _, err := action.GitHub.Repositories.Get(context.Background(), "actions", "toolkit")
	assert.NoError(t, err)
	assert.NotNil(t, repo.Owner)
	assert.NotNil(t, repo.Owner.Login)
	assert.EqualValues(t, "actions", *repo.Owner.Login)
}

func TestDownload(t *testing.T) {
	matcher := func(path string) bool {
		return path == "module.go"
	}

	files, err := action.DownloadSelectedRepositoryFiles(http.DefaultClient, "actions-go", "toolkit", "09edac1c7d93e0dd7fe5a14dc410fb0b41ea01c4", matcher)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "module.go", files["module.go"].Path)
	assert.Equal(t, []byte(content), files["module.go"].Data)
}
