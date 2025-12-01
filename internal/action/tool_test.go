package action_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/koblas/mushu/internal/action"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestTool(t *testing.T) *action.DownloadTool {
	t.Helper()

	rooter := func(path string) *os.Root {
		p := filepath.Join(t.TempDir(), path)
		err := os.Mkdir(p, 0755)
		require.NoError(t, err)
		r, err := os.OpenRoot(p)
		require.NoError(t, err)

		return r
	}

	return &action.DownloadTool{
		TmpRoot:   rooter("_temp"),
		CacheRoot: rooter("_cache"),
	}
}

func TestDownloadTool(t *testing.T) {
	data := "hello-world"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(data)) }))
	defer s.Close()

	d := buildTestTool(t)

	source, err := d.Download(s.URL)
	require.NoError(t, err)
	_, err = source.Root.Stat(source.Name)
	require.NoError(t, err)
	bytes, err := source.Root.ReadFile(source.Name)
	require.NoError(t, err)
	assert.Equal(t, data, string(bytes))
}

func TestGetCachedToolOrDownload(t *testing.T) {
	data := "hello-world"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(data)) }))
	defer s.Close()

	d := buildTestTool(t)

	f, err := d.GetCachedToolOrDownload(action.CacheOptions{Tool: "my-tool", Version: "1.0.1"}, s.URL)
	require.NoError(t, err)
	bytes, err := os.ReadFile(f)
	require.NoError(t, err)
	assert.Equal(t, data, string(bytes))

	s.Close()

	f, err = d.GetCachedToolOrDownload(action.CacheOptions{Tool: "my-tool", Version: "1.0.1"}, s.URL)
	require.NoError(t, err)
	bytes, err = os.ReadFile(f)
	require.NoError(t, err)
	assert.Equal(t, data, string(bytes))
}
