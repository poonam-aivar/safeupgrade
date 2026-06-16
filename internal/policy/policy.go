package policy

import (
	"fmt"
	"os"
	"strings"

	"github.com/aivar-tech/safeupgrade-agent/internal/scanner"
	"gopkg.in/yaml.v3"
)

type Policy struct {
	Global   GlobalPolicy             `yaml:"global"`
	Packages map[string]PackagePolicy `yaml:"packages"`
	Alerts   AlertConfig              `yaml:"alerts,omitempty"`
}

type GlobalPolicy struct {
	BlockCanary        bool     `yaml:"block_canary"`
	BlockAlpha         bool     `yaml:"block_alpha"`
	EnforceLockfile    bool     `yaml:"enforce_lockfile"`
	MaxMajorJump       int      `yaml:"max_major_jump"`
	ProvenanceRequired bool     `yaml:"provenance_required"`
	BlockedVersions    []string `yaml:"blocked_versions,omitempty"`
}

type PackagePolicy struct {
	PinMajor      int      `yaml:"pin_major,omitempty"`
	BlockCanary   bool     `yaml:"block_canary,omitempty"`
	BlockMajor    bool     `yaml:"block_major,omitempty"`
	AllowMinor    bool     `yaml:"allow_minor,omitempty"`
	BlockVersions []string `yaml:"block_versions,omitempty"`
	Reason        string   `yaml:"reason,omitempty"`
}

type AlertConfig struct {
	CompromisedPackage *AlertRule `yaml:"compromised_package,omitempty"`
	MaintainerChange   *AlertRule `yaml:"maintainer_change,omitempty"`
}

type AlertRule struct {
	Channel      string `yaml:"channel"`
	BypassPolicy bool   `yaml:"bypass_policy"`
	Action       string `yaml:"action"`
}

type Violation struct {
	Package string `json:"package"`
	Reason  string `json:"reason"`
}

func Load(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing policy YAML: %w", err)
	}
	return &p, nil
}

func Default() *Policy {
	return &Policy{
		Global: GlobalPolicy{
			BlockCanary:  true,
			BlockAlpha:   true,
			MaxMajorJump: 0,
		},
	}
}

func (p *Policy) Check(deps []scanner.Dependency) []Violation {
	var violations []Violation
	for _, dep := range deps {
		if v := p.checkDep(dep); v != nil {
			violations = append(violations, *v)
		}
	}
	return violations
}

func (p *Policy) Filter(deps []scanner.Dependency) []scanner.Dependency {
	var allowed []scanner.Dependency
	for _, dep := range deps {
		if !dep.Outdated {
			continue
		}
		if v := p.checkDep(dep); v == nil {
			allowed = append(allowed, dep)
		}
	}
	return allowed
}

func (p *Policy) checkDep(dep scanner.Dependency) *Violation {
	// Check global blocked versions
	for _, blocked := range p.Global.BlockedVersions {
		if dep.Latest == blocked {
			return &Violation{Package: dep.Name, Reason: fmt.Sprintf("version %s is globally blocked", dep.Latest)}
		}
	}

	// Check package-specific rules
	if pkg, ok := p.Packages[dep.Name]; ok {
		// Blocked versions
		for _, blocked := range pkg.BlockVersions {
			if dep.Latest == blocked {
				return &Violation{Package: dep.Name, Reason: fmt.Sprintf("version %s blocked: %s", dep.Latest, pkg.Reason)}
			}
		}

		// Pin major
		if pkg.PinMajor > 0 {
			latestMajor := getMajor(dep.Latest)
			if latestMajor > pkg.PinMajor {
				return &Violation{Package: dep.Name, Reason: fmt.Sprintf("pinned to major %d: %s", pkg.PinMajor, pkg.Reason)}
			}
		}

		// Block major
		if pkg.BlockMajor && getMajor(dep.Current) != getMajor(dep.Latest) {
			return &Violation{Package: dep.Name, Reason: "major upgrade blocked by policy"}
		}
	}

	// Global: block canary/alpha
	if p.Global.BlockCanary && isPrerelease(dep.Latest, "canary", "rc") {
		return &Violation{Package: dep.Name, Reason: "canary/rc version blocked by global policy"}
	}
	if p.Global.BlockAlpha && isPrerelease(dep.Latest, "alpha", "beta") {
		return &Violation{Package: dep.Name, Reason: "alpha/beta version blocked by global policy"}
	}

	// Global: max major jump
	if p.Global.MaxMajorJump == 0 && getMajor(dep.Current) != getMajor(dep.Latest) {
		return &Violation{Package: dep.Name, Reason: "major upgrade blocked (max_major_jump: 0)"}
	}

	return nil
}

func getMajor(version string) int {
	v := strings.TrimLeft(version, "^~>=<v")
	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return 0
	}
	major := 0
	fmt.Sscanf(parts[0], "%d", &major)
	return major
}

func isPrerelease(version string, tags ...string) bool {
	lower := strings.ToLower(version)
	for _, tag := range tags {
		if strings.Contains(lower, tag) {
			return true
		}
	}
	return false
}
