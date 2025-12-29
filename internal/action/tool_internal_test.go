package action

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestTool(t *testing.T) *DownloadTool {
	t.Helper()

	rooter := func(path string) *os.Root {
		p := filepath.Join(t.TempDir(), path)
		err := os.Mkdir(p, 0o755) // #nosec G301 -- test code
		require.NoError(t, err)

		r, err := os.OpenRoot(p)
		require.NoError(t, err)

		return r
	}

	return NewDownloadTool(
		WithDownloadTmpRoot(rooter("_temp")),
		WithDownloadCacheRoot(rooter("_cache")),
	)
}

func TestEnsureDestDir(t *testing.T) {
	d := buildTestTool(t)

	require.NoError(t, d.ensureDestDir(""))

	dir := uuid.New().String()

	assert.NoError(t, d.ensureDestDir(filepath.Join(dir, "some-file")))
	assert.NoError(t, d.ensureDestDir(filepath.Join(dir, "some-file")))
}

func TestEnsureDestNotExists(t *testing.T) {
	d := buildTestTool(t)

	name := uuid.New().String()
	fd, err := d.CacheRoot.Create(name)
	require.NoError(t, err)
	_ = fd.Close()

	require.NoError(t, d.ensureDestNotExists("sone-non-existing-file"))
	assert.Error(t, d.ensureDestNotExists(name))
}

func TestToolPath(t *testing.T) {
	options := CacheOptions{
		Tool:    "some-tool",
		Version: "version",
		Arch:    "arch",
	}
	path, err := options.path()
	require.NoError(t, err)
	assert.Equal(
		t,
		filepath.Join("some-tool", "version", "arch"),
		path,
	)
}

func TestCacheFile(t *testing.T) {
	d := buildTestTool(t)

	root, err := os.OpenRoot(".")
	require.NoError(t, err)

	path, err := d.CacheFile(&Source{Root: root, Name: "tool.go"}, "my-tool.go", CacheOptions{})
	require.Error(t, err)
	assert.Empty(t, path)

	path, err = d.CacheFile(&Source{Root: root, Name: "tool.go"}, "my-tool.go", CacheOptions{Tool: "some-tool", Version: "0.1.0"})
	require.NoError(t, err)
	// assert.Equal(t, filepath.Join(cacheRoot, "some-tool", "0.1.0"), path)
	assert.FileExists(t, path)

	/*
		Not sure what to do here

		path, err = d.CacheFile(&Source{Root: root, Name: "."}, "source", CacheOptions{Tool: "some-other-tool", Version: "0.1.0"})
		assert.NoError(t, err)
		// assert.Equal(t, filepath.Join(cacheRoot, "some-other-tool", "0.1.0"), path)
		assert.FileExists(t, filepath.Join(path, "source", "tool.go"))
		assert.FileExists(t, path+".complete")
	*/
}

// func TestListAllCachedVersions(t *testing.T) {
// 	d := buildTestTool(t)

// 	versions, err := d.ListAllCachedVersions(CacheOptions{Tool: "some-tool"})
// 	require.NoError(t, err)
// 	assert.Equal(t, []string{}, versions)
// 	os.MkdirAll(filepath.Join("some-tool", "1.203.2", "x86"), cachePerms)
// 	os.MkdirAll(filepath.Join("some-tool", "1.204.2", "x86"), cachePerms)
// 	os.MkdirAll(filepath.Join("some-tool", "1.205.2", "386"), cachePerms)
// 	versions, err = d.ListAllCachedVersions(CacheOptions{Tool: "some-tool"})
// 	require.NoError(t, err)
// 	assert.Len(t, versions, 3)
// 	assert.Contains(t, versions, "1.203.2")
// 	assert.Contains(t, versions, "1.204.2")
// 	assert.Contains(t, versions, "1.205.2")

// 	versions, err = d.ListAllCachedVersions(CacheOptions{Tool: "some-tool", Arch: "386"})
// 	require.NoError(t, err)
// 	assert.Len(t, versions, 1)
// 	assert.Contains(t, versions, "1.205.2")
// }

// func TestFindVersions(t *testing.T) {
// 	d := buildTestTool(t)

// 	os.MkdirAll(filepath.Join(cacheRoot, "some-tool", "1.203.2", "x86"), cachePerms)
// 	os.MkdirAll(filepath.Join(cacheRoot, "some-tool", "1.204.2", "x86"), cachePerms)
// 	os.MkdirAll(filepath.Join(cacheRoot, "some-tool", "1.205.2", "386"), cachePerms)
// 	os.MkdirAll(filepath.Join(cacheRoot, "some-tool", "1.205.2", "x86"), cachePerms)
// 	os.MkdirAll(filepath.Join(cacheRoot, "some-tool", "1.205.3", "x86"), cachePerms)

// 	path, err := d.FindVersion(CacheOptions{Tool: "some-tool"})
// 	assert.Error(t, err)
// 	assert.Equal(t, "", path)

// 	path, err = d.FindVersion(CacheOptions{Tool: "some-tool", Version: "~1.205"})
// 	assert.NoError(t, err)
// 	assert.Equal(t, filepath.Join(cacheRoot, "some-tool", "1.205.3"), path)

// 	path, err = d.FindVersion(CacheOptions{Tool: "some-tool", Version: "~1.205", Arch: "386"})
// 	assert.NoError(t, err)
// 	assert.Equal(t, filepath.Join(cacheRoot, "some-tool", "1.205.2", "386"), path)
// }

// func TestCacheDir(t *testing.T) {
// 	d := buildTestTool(t)

// 	root, err := os.OpenRoot("..")
// 	require.NoError(t, err)

// 	path, err := d.CacheDir(&Source{Root: root, Name: "cache"}, CacheOptions{Tool: "some-other-tool", Version: "0.1.0"})
// 	assert.NoError(t, err)
// 	// assert.Equal(t, filepath.Join(cacheRoot, "some-other-tool", "0.1.0"), path)
// 	assert.FileExists(t, filepath.Join(path, "tool.go"))
// }

func TestCopyURL(t *testing.T) {
	data := "hello-world"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(data)) }))
	defer s.Close()
	b := bytes.NewBuffer(nil)
	d := NewDownloadTool()
	require.NoError(t, d.copyURL(t.Context(), b, s.URL))
	assert.Equal(t, data, b.String())

	require.Error(t, d.copyURL(t.Context(), nil, s.URL))

	require.Error(t, d.copyURL(t.Context(), b, "this is not a URL"))

	require.Error(t, d.copyURL(t.Context(), writerInError{}, s.URL))

	s.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotAcceptable) })
	assert.Error(t, d.copyURL(t.Context(), bytes.NewBuffer(nil), s.URL))
}

type writerInError struct{}

func (writerInError) Write([]byte) (int, error) {
	return 0, errors.New("test-error")
}
