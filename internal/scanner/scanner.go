package scanner

import (
	"encoding/json"
	"fmt"
	"os/exec"
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
	cmd := exec.Command("npm", "outdated", "--json")
	cmd.Dir = s.repo
	out, _ := cmd.Output() // npm outdated exits 1 when deps are outdated

	var npmOut map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}

	if len(out) > 0 {
		if err := json.Unmarshal(out, &npmOut); err != nil {
			return nil, fmt.Errorf("parsing npm outdated: %w", err)
		}
	}

	deps := make([]Dependency, 0, len(npmOut))
	outdated := 0
	for name, info := range npmOut {
		isOutdated := info.Current != info.Latest
		if isOutdated {
			outdated++
		}
		deps = append(deps, Dependency{
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
		Total:        len(deps),
		Outdated:     outdated,
		Dependencies: deps,
	}, nil
}

func (s *Scanner) scanPip() (*Report, error) {
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

	deps := make([]Dependency, 0, len(pipOut))
	for _, p := range pipOut {
		deps = append(deps, Dependency{
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
		Total:        len(deps),
		Outdated:     len(deps),
		Dependencies: deps,
	}, nil
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
