package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aivar-tech/safeupgrade-agent/internal/scanner"
	"github.com/google/go-github/v62/github"
)

func executeScanOrg() error {
	if githubOrg == "" {
		return fmt.Errorf("--org flag is required for scan-org")
	}

	token := githubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("--github-token or GITHUB_TOKEN env required")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	fmt.Printf("  Org: %s\n", githubOrg)
	fmt.Println("  Fetching repositories...")

	var allRepos []*github.Repository
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(context.Background(), githubOrg, opts)
		if err != nil {
			return fmt.Errorf("listing repos: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Printf("  Found %d repositories\n\n", len(allRepos))

	tmpDir, err := os.MkdirTemp("", "safeupgrade-org-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	type result struct {
		Repo     string
		Lang     string
		Total    int
		Outdated int
	}
	var results []result

	for _, repo := range allRepos {
		if repo.GetArchived() {
			continue
		}

		name := repo.GetName()
		cloneURL := repo.GetCloneURL()
		repoDir := filepath.Join(tmpDir, name)

		cmd := exec.Command("git", "clone", "--depth=1", cloneURL, repoDir)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_ASKPASS=echo"), fmt.Sprintf("GIT_TERMINAL_PROMPT=0"))
		if _, err := cmd.CombinedOutput(); err != nil {
			continue
		}

		lang := detectLanguage(repoDir, "")
		if lang == "" {
			continue
		}

		fmt.Printf("  📦 %s (%s)... ", name, lang)

		s, err := scanner.New(lang, repoDir)
		if err != nil {
			fmt.Printf("skip\n")
			continue
		}

		report, err := s.Scan()
		if err != nil {
			fmt.Printf("skip\n")
			continue
		}

		fmt.Printf("%d outdated\n", report.Outdated)
		results = append(results, result{Repo: name, Lang: lang, Total: report.Total, Outdated: report.Outdated})
	}

	fmt.Printf("\n📊 Org Scan Summary:\n")
	fmt.Printf("  %-30s %-8s %-8s %-8s\n", "REPO", "LANG", "TOTAL", "OUTDATED")
	fmt.Printf("  %-30s %-8s %-8s %-8s\n", "----", "----", "-----", "--------")
	totalOutdated := 0
	for _, r := range results {
		fmt.Printf("  %-30s %-8s %-8d %-8d\n", r.Repo, r.Lang, r.Total, r.Outdated)
		totalOutdated += r.Outdated
	}
	fmt.Printf("\n  Total: %d repos scanned, %d outdated dependencies\n", len(results), totalOutdated)

	return nil
}
