package action

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v79/github"
	"golang.org/x/oauth2"
)

func token() string {
	if v, ok := os.LookupEnv("GITHUB_TOKEN"); ok && v != "" {
		return v
	}

	for _, input := range []string{"github-token", "token"} {
		if t, ok := GetInput(input); ok {
			return t
		}
	}

	return ""
}

func NewClient(ctx context.Context) (*github.Client, error) {
	token := token()
	httpClient := http.DefaultClient
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	}
	client := github.NewClient(httpClient)
	if server, ok := os.LookupEnv("GITHUB_SERVER_URL"); ok {
		var err error
		client, err = client.WithEnterpriseURLs(server, server)
		if err != nil {
			return nil, fmt.Errorf("configure GitHub Enterprise URLs: %w", err)
		}
	}

	return client, nil
}

func authorize(r *http.Request) {
	t := token()
	if t != "" {
		r.SetBasicAuth("", t)
	}
}

type Matcher func(path string) bool

type RepositoryFile struct {
	Path     string
	FileInfo os.FileInfo
	Data     []byte
}

// DownloadSelectedRepositoryFiles downloads files from a given repository and granch, given that their name matches regarding the `include` function
func DownloadSelectedRepositoryFiles(ctx context.Context, c *http.Client, owner, repo, branch string, include Matcher) (map[string]RepositoryFile, error) {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", owner, repo, branch)
	slog.Debug("Downloading tarball for", "repo", u)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request to download repository: %w", err)
	}
	authorize(req)
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not download repository: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not download repository: unexpected code %d", resp.StatusCode)
	}
	defer func() { _ = resp.Body.Close() }()
	var body io.Reader = resp.Body
	switch resp.Header.Get("Content-Type") {
	case "application/gzip", "application/x-gzip":
		body, err = gzip.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("could not create gzip reader: %w", err)
		}
	}
	files := map[string]RepositoryFile{}
	tr := tar.NewReader(body)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("could not read tarball: %w", err)
		}
		if hdr.Format == tar.FormatPAX || hdr.FileInfo().IsDir() {
			continue
		}
		splittedName := strings.SplitN(hdr.Name, "/", 2)
		if len(splittedName) > 1 {
			name := splittedName[1]
			if include(name) {
				slog.Debug("Downloading", slog.String("name", hdr.Name))
				b := bytes.NewBuffer(nil)
				_, err := io.Copy(b, tr) // #nosec G110 -- this chould be fixed later
				if err != nil {
					return nil, fmt.Errorf("could not read file %q from tarball: %w", name, err)
				}
				files[name] = RepositoryFile{
					Path:     name,
					FileInfo: hdr.FileInfo(),
					Data:     b.Bytes(),
				}
			}
		}
	}

	return files, nil
}
