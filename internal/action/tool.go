package action

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
)

const (
	cachePerms = 0755
)

// DownloadToolOptions defines available options to download tools
type DownloadTool struct {
	// RUNNER_TEMP
	TmpRoot *os.Root
	// RUNNER_TOOL_CACHE
	CacheRoot *os.Root

	Destination string
	FileMode    os.FileMode
}

// CacheOptions defines the available options for tool and file caching
type CacheOptions struct {
	Tool    string
	Version string
	Arch    string
	// UseJavascriptValues instructs to use
	// the javascript os.arch() and os.platform() values instead of respectively
	// runtime.GOARCH and runtime.GOOS
	UseJavascriptValues *bool
}

func Bool(b bool) *bool {
	return &b
}

func (d *DownloadTool) destination() string {
	return uuid.New().String()
}

var (
	tmpRoot   *os.Root
	cacheRoot *os.Root
)

func (d *DownloadTool) getTmpRoot() *os.Root {
	if d.TmpRoot != nil {
		return d.TmpRoot
	}

	if tmpRoot == nil {
		r, err := os.OpenRoot(tempDirectory)
		if err != nil {
			panic(fmt.Sprintf("unable to open root: %v", err))
		}
		tmpRoot = r
	}

	return tmpRoot
}

func (d *DownloadTool) getCacheRoot() *os.Root {
	if d.CacheRoot != nil {
		return d.CacheRoot
	}

	if cacheRoot == nil {
		r, err := os.OpenRoot(cacheDirectory)
		if err != nil {
			panic(fmt.Sprintf("unable to open root: %v", err))
		}
		cacheRoot = r
	}

	return cacheRoot
}

func (d *DownloadTool) ensureDestDir(dest string) error {
	destDir := filepath.Dir(dest)
	if destDir == "" {
		return nil
	}

	root := d.getCacheRoot()

	if err := root.MkdirAll(destDir, cachePerms); err != nil {
		return fmt.Errorf("unable to create destination directory %s: %v", destDir, err)
	}
	return nil
}

func (d *DownloadTool) createEmptyCache(folder string) error {
	root := d.getCacheRoot()

	if err := root.RemoveAll(folder); err != nil {
		return err
	}

	return root.MkdirAll(folder, cachePerms)
}

func (d *DownloadTool) ensureDestNotExists(dest string) error {
	root := d.getCacheRoot()

	_, err := root.Stat(dest)
	if err == nil {
		return fmt.Errorf("already exists")
	}
	if !os.IsNotExist(err) {
		return err
	}

	return nil
}

func copyURL(dest io.Writer, source string) error {
	wrapError := func(err error, format string, args ...interface{}) error {
		return fmt.Errorf("failed to download "+source+" "+format+" : %v", append(args, err)...)
	}
	if dest == nil {
		return wrapError(fmt.Errorf("destination should not be null"), "")
	}
	resp, err := http.Get(source)
	if err != nil {
		return wrapError(err, "download failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return wrapError(fmt.Errorf("unexpected status code %d (%s). Expecting %d", resp.StatusCode, resp.Status, http.StatusOK), "")
	}
	_, err = io.Copy(dest, resp.Body)
	if err != nil {
		return wrapError(err, "failed to write to destination")
	}
	return nil
}

func jsArch() string {
	// mapping https://github.com/golang/go/blob/98d2717499575afe13d9f815d46fcd6e384efb0c/src/go/build/syslist.go#L11
	// to https://nodejs.org/api/os.html#os_os_arch
	switch runtime.GOARCH {
	case "386":
		return "x32"
	case "amd64":
		return "x64"
	case "arm":
		return "arm"
	case "arm64":
		return "arm64"
	case "ppc64", "ppc64le":
		return "ppc64"
	case "mipsle":
		return "mipsel"
	case "mips64le":
		return "mips64el"
	default:
		return runtime.GOARCH
	}
}

func (c CacheOptions) arch() string {
	if c.Arch != "" {
		return c.Arch
	}

	if c.UseJavascriptValues == nil || *c.UseJavascriptValues {
		return jsArch()
	}

	return runtime.GOARCH
}

func (c CacheOptions) version() string {
	return strings.TrimPrefix(c.Version, "=v")
}

func (c CacheOptions) path() (string, error) {
	if c.Tool == "" {
		return "", fmt.Errorf("missing tool name in options.Tool")
	}

	return filepath.Join(c.Tool, c.version(), c.arch()), nil
}

type Source struct {
	Root *os.Root
	Name string
}

func (d *DownloadTool) cache(source *Source, target string, options CacheOptions) (string, error) {
	wrapError := func(err error, format string, args ...interface{}) (string, error) {
		return "", fmt.Errorf("failed to save "+source.Name+" to cache "+format+" : %v", append(args, err)...)
	}

	destFolder, err := options.path()
	if err != nil {
		return wrapError(err, "")
	}
	completeMarker := destFolder + ".complete"

	// Cleanup any prior incomplete cache
	slog.Debug("destination", "file", destFolder)
	err = d.createEmptyCache(destFolder)
	if err != nil {
		return wrapError(err, "")
	}
	err = os.Remove(completeMarker)
	if err != nil && !os.IsNotExist(err) {
		return wrapError(err, "")
	}

	// Ensure provided arguments are namespaced to the destFolder
	spath := filepath.Join(source.Root.Name(), source.Name)
	err = filepath.WalkDir(spath, func(path string, info os.DirEntry, err error) error {
		if info == nil || info.IsDir() || err != nil {
			return nil
		}

		destination := filepath.Join(destFolder, target)

		slog.Debug("copying", "src", path, "dest", destination)

		return d.copyToCache(destination, path)
	})
	if err != nil {
		return wrapError(err, "copy all files")
	}
	root := d.getCacheRoot()
	fd, err := root.Create(completeMarker)
	if err != nil {
		return wrapError(err, "mark copy complete")
	}
	_ = fd.Close()

	return filepath.Join(root.Name(), destFolder), nil
}

func (d *DownloadTool) copyToCache(dest, src string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source file %s: %v", src, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %v", src, err)
	}

	root := d.getCacheRoot()
	err = root.MkdirAll(filepath.Dir(dest), cachePerms)
	if err != nil {
		return fmt.Errorf("make dest directory %s: %v", filepath.Dir(dest), err)
	}
	out, err := root.Create(dest)
	if err != nil {
		return fmt.Errorf("create dest file %s: %v", dest, err)
	}
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("copy data: %w", err)
	}
	err = root.Chmod(dest, stat.Mode())
	if err != nil {
		return fmt.Errorf("chmod dest file %s: %v", dest, err)
	}

	return nil
}

// CacheFile caches a downloaded file (GUID) and installs it
// into the tool cache with a given targetName
func (d *DownloadTool) CacheFile(source *Source, target string, options CacheOptions) (string, error) {
	path, err := d.cache(source, target, options)
	if err != nil {
		return "", err
	}

	return filepath.Join(path, target), nil
}

// CacheDir caches a directory and installs it into the tool cacheDir
// with a given targetName
func (d *DownloadTool) CacheDir(source *Source, options CacheOptions) (string, error) {
	return d.cache(source, "", options)
}

// ListAllCachedVersions discovers all versions available in cache
func (d *DownloadTool) ListAllCachedVersions(options CacheOptions) ([]string, error) {
	if options.Tool == "" {
		return nil, fmt.Errorf("missing tool name to list versions in options.Tool")
	}

	root := d.getCacheRoot()
	versions := []string{}

	// Check if tool folder exists
	if _, err := root.Stat(options.Tool); os.IsNotExist(err) {
		return versions, nil
	}

	rfs := root.FS()
	rdir, ok := rfs.(fs.ReadDirFS)
	if !ok {
		return nil, fmt.Errorf("unable to read cache directory for tool %s", options.Tool)
	}

	dirs, err := rdir.ReadDir(options.Tool)
	if err != nil {
		return nil, fmt.Errorf("read cache directory for tool %s: %v", options.Tool, err)
	}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		path := filepath.Join(options.Tool, dir.Name(), options.Arch)
		info, err := root.Stat(path)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}

		versions = append(versions, dir.Name())
	}

	return versions, nil
}

// FindVersion finds the path to a tool version in the local installed tool cache
// by matching the pattern provided in the Version field
func (d *DownloadTool) FindVersion(options CacheOptions) (string, error) {
	versions := semver.Collection{}
	constraint, err := semver.NewConstraint(options.Version)
	if err != nil {
		return "", fmt.Errorf("invalid version constraint %s: %v", options.Version, err)
	}
	vinfo, err := d.ListAllCachedVersions(options)
	if err != nil {
		return "", fmt.Errorf("list cached versions for tool %s: %v", options.Tool, err)
	}
	for _, v := range vinfo {
		s, err := semver.NewVersion(v)
		if err == nil {
			versions = append(versions, s)
		} else {
			slog.Debug("invalid semver from cache", "version", v, "error", err)
		}
	}

	root := d.getCacheRoot()
	sort.Sort(versions)
	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		if constraint.Check(version) {
			options.Version = version.Original()
			path, err := options.path()
			if err != nil {
				return "", err
			}
			path = filepath.Join(path, options.Tool)
			if _, err := root.Stat(path); err == nil {
				return filepath.Join(root.Name(), path), nil
			}
		}
	}

	return "", fmt.Errorf("could not find any cached version for %s matching %s", options.Tool, options.Version)
}

// DownloadTool Download a tool from an url and stream it into a file
func (d *DownloadTool) Download(url string) (*Source, error) {
	root := d.getTmpRoot()

	wrapError := func(err error, format string, args ...interface{}) (*Source, error) {
		return nil, fmt.Errorf(format+" : %v", append(args, err)...)
	}
	dest := d.destination()
	slog.Debug("downloading", "url", url, "dest", dest)
	if err := d.ensureDestDir(dest); err != nil {
		return wrapError(err, "Unable to create destination directory")
	}
	if err := d.ensureDestNotExists(dest); err != nil {
		return wrapError(err, "Destination file path %v", dest)
	}
	out, err := root.Create(dest)
	if err != nil {
		return wrapError(err, "failed to create destination file %s", dest)
	}
	if err := copyURL(out, url); err != nil {
		return wrapError(err, "failed to write file %s", dest)
	}
	if d.FileMode != 0 {
		err = root.Chmod(dest, d.FileMode)
		if err != nil {
			return wrapError(err, "failed to chmod file %s", dest)
		}
	}

	return &Source{
		Root: root,
		Name: dest,
	}, nil
}

// GetCachedToolOrDownload returns the path of a cached tool or downloads and caches it
func (d *DownloadTool) GetCachedToolOrDownload(cache CacheOptions, url string) (string, error) {
	if path, err := d.FindVersion(cache); err == nil {
		return path, nil
	}

	source, err := d.Download(url)
	if err != nil {
		return "", err
	}
	path, err := d.CacheFile(source, cache.Tool, cache)
	if err != nil {
		return "", fmt.Errorf("failed to cache downloaded tool %v: %v", url, err)
	}

	return path, nil
}
