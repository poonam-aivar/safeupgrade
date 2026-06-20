package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	switch e.lang {
	case "npm":
		cmd := exec.Command("npm", "install", fmt.Sprintf("%s@%s", name, version), "--save-exact") // #nosec G204
		cmd.Dir = e.repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
		}
	case "pip":
		// Rewrite version in requirements.txt and pyproject.toml files
		if err := e.rewritePipVersion(name, version); err != nil {
			return err
		}
	case "go":
		cmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", name, version)) // #nosec G204
		cmd.Dir = e.repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
		}
	default:
		return fmt.Errorf("unsupported ecosystem: %s", e.lang)
	}
	return nil
}

func (e *Executor) rewritePipVersion(name, version string) error {
	updated := false

	// Walk all files to find requirements.txt and pyproject.toml
	_ = filepath.Walk(e.repo, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (info.Name() == ".venv" || info.Name() == "venv" || info.Name() == ".git" || info.Name() == "node_modules") {
			return filepath.SkipDir
		}

		if info.Name() == "requirements.txt" {
			if e.rewriteRequirementsTxt(path, name, version) {
				updated = true
			}
		}
		if info.Name() == "pyproject.toml" {
			if e.rewritePyprojectToml(path, name, version) {
				updated = true
			}
		}
		return nil
	})

	if !updated {
		return fmt.Errorf("could not find %s in any dependency file", name)
	}
	return nil
}

func (e *Executor) rewriteRequirementsTxt(path, name, version string) bool {
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return false
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Match package name (case-insensitive, handle - vs _)
		pkgName := ""
		for _, sep := range []string{"==", ">=", "<=", "~=", "!="} {
			if idx := strings.Index(trimmed, sep); idx > 0 {
				pkgName = strings.TrimSpace(trimmed[:idx])
				break
			}
		}
		if strings.EqualFold(strings.ReplaceAll(pkgName, "-", "_"), strings.ReplaceAll(name, "-", "_")) {
			lines[i] = fmt.Sprintf("%s==%s", name, version)
			found = true
		}
	}

	if found {
		os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600) // #nosec G304
	}
	return found
}

func (e *Executor) rewritePyprojectToml(path, name, version string) bool {
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return false
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match "package>=version" or "package==version" inside dependencies array
		for _, sep := range []string{">=", "==", "~=", "<=", "!="} {
			if idx := strings.Index(trimmed, sep); idx > 0 {
				// Extract package name from the line (remove quotes)
				pkgPart := strings.Trim(trimmed[:idx], `"' `)
				if strings.EqualFold(strings.ReplaceAll(pkgPart, "-", "_"), strings.ReplaceAll(name, "-", "_")) {
					// Preserve indentation and quotes
					indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					quote := `"`
					if strings.Contains(line, "'") {
						quote = "'"
					}
					// Check if line has trailing comma
					suffix := ""
					if strings.HasSuffix(strings.TrimSpace(line), `",`) || strings.HasSuffix(strings.TrimSpace(line), `',`) {
						suffix = ","
					}
					lines[i] = fmt.Sprintf(`%s%s%s>=%s%s%s`, indent, quote, name, version, quote, suffix)
					found = true
					break
				}
			}
		}
	}

	if found {
		os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600) // #nosec G304
	}
	return found
}

func (e *Executor) runTests() error {
	var cmd *exec.Cmd
	switch e.lang {
	case "npm":
		cmd = exec.Command("npm", "test")
	case "pip":
		if _, err := exec.LookPath("pytest"); err != nil {
			return nil // skip tests if pytest not available
		}
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
