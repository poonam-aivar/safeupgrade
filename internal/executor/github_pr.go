package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v62/github"
)

type PRResult struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// CreatePR creates a GitHub pull request with the upgrade report as the body.
func (e *Executor) CreatePR(token, branch, title, body string) (*PRResult, error) {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("no GitHub token provided (use --github-token or GITHUB_TOKEN env)")
	}

	owner, repo, err := getRepoInfo(e.repo)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(nil).WithAuthToken(token)
	base := "main"

	pr, _, err := client.PullRequests.Create(context.Background(), owner, repo, &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &branch,
		Base:  &base,
	})
	if err != nil {
		return nil, fmt.Errorf("creating PR: %w", err)
	}

	return &PRResult{
		URL:    pr.GetHTMLURL(),
		Number: pr.GetNumber(),
		Title:  pr.GetTitle(),
	}, nil
}

func getRepoInfo(repoPath string) (string, string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("getting git remote: %w", err)
	}

	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")
	url = strings.Replace(url, "git@github.com:", "", 1)
	url = strings.TrimPrefix(url, "https://github.com/")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot parse owner/repo from: %s", url)
	}
	return parts[0], parts[1], nil
}
