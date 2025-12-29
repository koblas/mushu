package action_test

import (
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
	client, err := action.NewClient(t.Context())
	require.NoError(t, err)

	repo, _, err := client.Repositories.Get(t.Context(), "actions", "toolkit")
	require.NoError(t, err)
	assert.NotNil(t, repo.Owner)
	assert.NotNil(t, repo.Owner.Login)
	assert.Equal(t, "actions", *repo.Owner.Login)
}

func TestDownload(t *testing.T) {
	matcher := func(path string) bool {
		return path == "module.go"
	}

	files, err := action.DownloadSelectedRepositoryFiles(t.Context(), http.DefaultClient, "actions-go", "toolkit", "09edac1c7d93e0dd7fe5a14dc410fb0b41ea01c4", matcher)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "module.go", files["module.go"].Path)
	assert.Equal(t, []byte(content), files["module.go"].Data)
}
