package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Changelog struct {
	Package string `json:"package"`
	Version string `json:"version"`
	Source  string `json:"source"` // "github" or "npm"
	Body    string `json:"body"`
	Date    string `json:"published_at"`
	URL     string `json:"url"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// FetchChangelog tries GitHub releases first, then falls back to npm registry.
func FetchChangelog(pkg, version string) (*Changelog, error) {
	repo := guessGitHubRepo(pkg)
	if repo != "" {
		if cl, err := fetchGitHubRelease(repo, version); err == nil {
			return cl, nil
		}
	}
	return fetchNpmChangelog(pkg, version)
}

func fetchGitHubRelease(repo, version string) (*Changelog, error) {
	// Try common tag formats: v1.2.3, 1.2.3, package@1.2.3
	tags := []string{"v" + version, version}
	parts := strings.Split(repo, "/")
	if len(parts) == 2 {
		tags = append(tags, parts[1]+"@"+version)
	}

	for _, tag := range tags {
		url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := httpClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				_ = resp.Body.Close()
			}
			continue
		}
		defer resp.Body.Close()

		var release struct {
			Body        string `json:"body"`
			PublishedAt string `json:"published_at"`
			HTMLURL     string `json:"html_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			continue
		}

		body := release.Body
		if len(body) > 3000 {
			body = body[:3000] + "\n...(truncated)"
		}

		return &Changelog{
			Package: repo,
			Version: version,
			Source:  "github",
			Body:    body,
			Date:    release.PublishedAt,
			URL:     release.HTMLURL,
		}, nil
	}
	return nil, fmt.Errorf("no GitHub release found for %s@%s", repo, version)
}

func fetchNpmChangelog(pkg, version string) (*Changelog, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s/%s", pkg, version)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("npm registry returned %d for %s@%s", resp.StatusCode, pkg, version)
	}

	body, _ := io.ReadAll(resp.Body)
	var meta struct {
		Description string            `json:"description"`
		Dist        map[string]string `json:"dist"`
		Repository  struct {
			URL string `json:"url"`
		} `json:"repository"`
	}
	_ = json.Unmarshal(body, &meta)

	return &Changelog{
		Package: pkg,
		Version: version,
		Source:  "npm",
		Body:    fmt.Sprintf("Package: %s@%s\nDescription: %s", pkg, version, meta.Description),
		URL:     fmt.Sprintf("https://www.npmjs.com/package/%s/v/%s", pkg, version),
	}, nil
}

// guessGitHubRepo maps npm package names to GitHub repos.
func guessGitHubRepo(pkg string) string {
	// Try npm registry to get repository URL
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkg)
	resp, err := httpClient.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var meta struct {
		Repository struct {
			URL string `json:"url"`
		} `json:"repository"`
	}
	json.NewDecoder(resp.Body).Decode(&meta) //nolint:errcheck // best-effort metadata lookup

	repoURL := meta.Repository.URL
	// Parse "git+https://github.com/owner/repo.git" or "https://github.com/owner/repo"
	repoURL = strings.TrimPrefix(repoURL, "git+")
	repoURL = strings.TrimPrefix(repoURL, "git://")
	repoURL = strings.TrimPrefix(repoURL, "https://github.com/")
	repoURL = strings.TrimPrefix(repoURL, "http://github.com/")
	repoURL = strings.TrimSuffix(repoURL, ".git")

	if strings.Count(repoURL, "/") >= 1 && !strings.Contains(repoURL, "://") {
		// Take only owner/repo (ignore subdirectories)
		parts := strings.SplitN(repoURL, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return ""
}
