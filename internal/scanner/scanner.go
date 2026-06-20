package scanner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Dependency struct {
	Name       string `json:"name"`
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Outdated   bool   `json:"outdated"`
	Vulnerable bool   `json:"vulnerable,omitempty"`
	CVEs       []string `json:"cves,omitempty"`
}

type Report struct {
	Project      string       `json:"project"`
	ScannedAt    time.Time    `json:"scanned_at"`
	Ecosystem    string       `json:"ecosystem"`
	Total        int          `json:"total"`
	Outdated     int          `json:"outdated"`
	Vulnerable   int          `json:"vulnerable"`
	Dependencies []Dependency `json:"dependencies"`
}

type Scanner struct {
	lang string
	repo string
}

func New(lang, repo string) (*Scanner, error) {
	return &Scanner{lang: lang, repo: repo}, nil
}

func (s *Scanner) Scan() (*Report, error) {
	switch s.lang {
	case "npm":
		return s.scanNpm()
	case "pip":
		return s.scanPip()
	case "go":
		return s.scanGo()
	default:
		return nil, fmt.Errorf("unsupported ecosystem: %s", s.lang)
	}
}

func (s *Scanner) scanNpm() (*Report, error) {
	// Try direct package.json parsing first (no install needed)
	deps, err := s.scanNpmDirect()
	if err == nil && len(deps) > 0 {
		outdated := 0
		for _, d := range deps {
			if d.Outdated {
				outdated++
			}
		}
		return &Report{
			Project:      s.repo,
			ScannedAt:    time.Now(),
			Ecosystem:    "npm",
			Total:        len(deps),
			Outdated:     outdated,
			Dependencies: deps,
		}, nil
	}

	// Fallback to npm outdated (needs node_modules)
	cmd := exec.Command("npm", "outdated", "--json")
	cmd.Dir = s.repo
	out, _ := cmd.Output()

	var npmOut map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}

	if len(out) > 0 {
		if err := json.Unmarshal(out, &npmOut); err != nil {
			return nil, fmt.Errorf("parsing npm outdated: %w", err)
		}
	}

	npmDeps := make([]Dependency, 0, len(npmOut))
	outdated := 0
	for name, info := range npmOut {
		isOutdated := info.Current != info.Latest
		if isOutdated {
			outdated++
		}
		npmDeps = append(npmDeps, Dependency{
			Name:     name,
			Current:  info.Current,
			Latest:   info.Latest,
			Outdated: isOutdated,
		})
	}

	return &Report{
		Project:      s.repo,
		ScannedAt:    time.Now(),
		Ecosystem:    "npm",
		Total:        len(npmDeps),
		Outdated:     outdated,
		Dependencies: npmDeps,
	}, nil
}

func (s *Scanner) scanNpmDirect() ([]Dependency, error) {
	// Find package.json files recursively
	var pkgFiles []string
	_ = filepath.Walk(s.repo, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist") {
			return filepath.SkipDir
		}
		if info.Name() == "package.json" {
			pkgFiles = append(pkgFiles, path)
		}
		return nil
	})

	if len(pkgFiles) == 0 {
		return nil, fmt.Errorf("no package.json found")
	}

	var deps []Dependency
	for _, pkgFile := range pkgFiles {
		data, err := os.ReadFile(pkgFile) // #nosec G304
		if err != nil {
			continue
		}

		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			continue
		}

		for name, version := range pkg.Dependencies {
			current := strings.TrimLeft(version, "^~>=<")
			latest := queryNpmRegistry(name)
			if latest == "" || latest == current {
				continue
			}
			deps = append(deps, Dependency{
				Name:     name,
				Current:  current,
				Latest:   latest,
				Outdated: true,
			})
		}
	}

	return deps, nil
}

func queryNpmRegistry(pkg string) string {
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/%s/latest", pkg))
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.Version
}

func (s *Scanner) scanPip() (*Report, error) {
	// Try parsing requirements.txt / pyproject.toml directly (no install needed)
	deps, err := s.scanPipDirect()
	if err == nil && len(deps) > 0 {
		outdated := 0
		for _, d := range deps {
			if d.Outdated {
				outdated++
			}
		}
		return &Report{
			Project:      s.repo,
			ScannedAt:    time.Now(),
			Ecosystem:    "pip",
			Total:        len(deps),
			Outdated:     outdated,
			Dependencies: deps,
		}, nil
	}

	// Fallback to pip list --outdated (needs deps installed)
	pipCmd := "pip"
	if _, err := exec.LookPath("pip"); err != nil {
		pipCmd = "pip3"
	}

	cmd := exec.Command(pipCmd, "list", "--outdated", "--format=json")
	cmd.Dir = s.repo
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running %s list: %w", pipCmd, err)
	}

	var pipOut []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Latest  string `json:"latest_version"`
	}

	if err := json.Unmarshal(out, &pipOut); err != nil {
		return nil, fmt.Errorf("parsing pip output: %w", err)
	}

	pipDeps := make([]Dependency, 0, len(pipOut))
	for _, p := range pipOut {
		pipDeps = append(pipDeps, Dependency{
			Name:     p.Name,
			Current:  p.Version,
			Latest:   p.Latest,
			Outdated: true,
		})
	}

	return &Report{
		Project:      s.repo,
		ScannedAt:    time.Now(),
		Ecosystem:    "pip",
		Total:        len(pipDeps),
		Outdated:     len(pipDeps),
		Dependencies: pipDeps,
	}, nil
}

func (s *Scanner) scanPipDirect() ([]Dependency, error) {
	var lines []string

	// Recursively find all requirements.txt and pyproject.toml files
	_ = filepath.Walk(s.repo, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".venv" || info.Name() == "venv" || info.Name() == ".git") {
			return filepath.SkipDir
		}
		if info.Name() == "requirements.txt" {
			data, err := os.ReadFile(path) // #nosec G304
			if err == nil {
				lines = append(lines, strings.Split(string(data), "\n")...)
			}
		}
		if info.Name() == "pyproject.toml" {
			data, err := os.ReadFile(path) // #nosec G304
			if err == nil {
				lines = append(lines, parsePyprojectDeps(string(data))...)
			}
		}
		return nil
	})

	if len(lines) == 0 {
		return nil, fmt.Errorf("no requirements.txt or pyproject.toml found")
	}

	var deps []Dependency
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		// Parse "package>=version" or "package==version"
		var name, currentVer string
		for _, sep := range []string{"==", ">=", "<=", "~=", "!="} {
			if idx := strings.Index(line, sep); idx > 0 {
				name = strings.TrimSpace(line[:idx])
				currentVer = strings.TrimSpace(line[idx+len(sep):])
				// Remove inline comments
				if ci := strings.Index(currentVer, "#"); ci > 0 {
					currentVer = strings.TrimSpace(currentVer[:ci])
				}
				break
			}
		}
		if name == "" {
			continue
		}

		// Query PyPI for latest version
		latest := queryPyPI(name)
		if latest == "" {
			continue
		}

		outdated := latest != currentVer
		deps = append(deps, Dependency{
			Name:     name,
			Current:  currentVer,
			Latest:   latest,
			Outdated: outdated,
		})
	}

	return deps, nil
}

func parsePyprojectDeps(content string) []string {
	var deps []string
	inDeps := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Look for dependencies section
		if trimmed == "dependencies = [" || trimmed == "[project.dependencies]" {
			inDeps = true
			continue
		}

		// End of array
		if inDeps && (trimmed == "]" || (strings.HasPrefix(trimmed, "[") && trimmed != "[")) {
			inDeps = false
			continue
		}

		if inDeps && trimmed != "" {
			// Clean up: remove quotes, commas, trailing comments
			dep := strings.Trim(trimmed, `"',`)
			dep = strings.TrimSpace(dep)
			if dep != "" && !strings.HasPrefix(dep, "#") {
				deps = append(deps, dep)
			}
		}
	}
	return deps
}

func queryPyPI(pkg string) string {
	resp, err := http.Get(fmt.Sprintf("https://pypi.org/pypi/%s/json", pkg))
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.Info.Version
}

func (s *Scanner) scanGo() (*Report, error) {
	cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
	cmd.Dir = s.repo
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running go list: %w", err)
	}

	type goMod struct {
		Path    string `json:"Path"`
		Version string `json:"Version"`
		Update  *struct {
			Version string `json:"Version"`
		} `json:"Update"`
	}

	var deps []Dependency
	outdated := 0

	// go list -json outputs concatenated JSON objects
	decoder := json.NewDecoder(strings.NewReader(string(out)))
	for decoder.More() {
		var m goMod
		if err := decoder.Decode(&m); err != nil {
			break
		}
		if m.Update != nil {
			outdated++
			deps = append(deps, Dependency{
				Name:     m.Path,
				Current:  m.Version,
				Latest:   m.Update.Version,
				Outdated: true,
			})
		}
	}

	return &Report{
		Project:      s.repo,
		ScannedAt:    time.Now(),
		Ecosystem:    "go",
		Total:        len(deps),
		Outdated:     outdated,
		Dependencies: deps,
	}, nil
}
