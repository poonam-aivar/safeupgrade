package executor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aivar-tech/safeupgrade-agent/internal/analyzer"
)

type UpgradeResult struct {
	Upgraded   int              `json:"upgraded"`
	Skipped    int              `json:"skipped"`
	TestsPassed bool           `json:"tests_passed"`
	Packages   []PackageResult `json:"packages"`
	ExecutedAt time.Time       `json:"executed_at"`
}

type PackageResult struct {
	Name   string `json:"name"`
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"` // upgraded, failed, skipped
	Error  string `json:"error,omitempty"`
}

type Executor struct {
	lang string
	repo string
}

func New(lang, repo string) *Executor {
	return &Executor{lang: lang, repo: repo}
}

func (e *Executor) Upgrade(safe []analyzer.Result) (*UpgradeResult, error) {
	result := &UpgradeResult{ExecutedAt: time.Now()}

	for _, dep := range safe {
		err := e.upgradeDep(dep.Package, dep.To)
		if err != nil {
			result.Skipped++
			result.Packages = append(result.Packages, PackageResult{
				Name:   dep.Package,
				From:   dep.From,
				To:     dep.To,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}
		result.Upgraded++
		result.Packages = append(result.Packages, PackageResult{
			Name:   dep.Package,
			From:   dep.From,
			To:     dep.To,
			Status: "upgraded",
		})
	}

	// Run tests
	if err := e.runTests(); err != nil {
		result.TestsPassed = false
		return result, fmt.Errorf("tests failed after upgrade: %w", err)
	}
	result.TestsPassed = true

	return result, nil
}

func (e *Executor) upgradeDep(name, version string) error {
	var cmd *exec.Cmd
	switch e.lang {
	case "npm":
		cmd = exec.Command("npm", "install", fmt.Sprintf("%s@%s", name, version), "--save-exact") // #nosec G204 -- name/version from package manager output
	case "pip":
		cmd = exec.Command("pip", "install", fmt.Sprintf("%s==%s", name, version)) // #nosec G204
	case "go":
		cmd = exec.Command("go", "get", fmt.Sprintf("%s@%s", name, version)) // #nosec G204
	default:
		return fmt.Errorf("unsupported ecosystem: %s", e.lang)
	}
	cmd.Dir = e.repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (e *Executor) runTests() error {
	var cmd *exec.Cmd
	switch e.lang {
	case "npm":
		cmd = exec.Command("npm", "test")
	case "pip":
		cmd = exec.Command("pytest")
	case "go":
		cmd = exec.Command("go", "test", "./...")
	default:
		return nil
	}
	cmd.Dir = e.repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (e *Executor) CreateBranch(branchName string) error {
	cmds := [][]string{
		{"git", "checkout", "-b", branchName},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...) // #nosec G204 -- args are hardcoded git commands
		cmd.Dir = e.repo
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
	}
	return nil
}

func (e *Executor) CommitAndPush(branchName, message string) error {
	cmds := [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", message},
		{"git", "push", "-u", "origin", branchName},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...) // #nosec G204 -- args are hardcoded git commands
		cmd.Dir = e.repo
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
	}
	return nil
}
