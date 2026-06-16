package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Vulnerability struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Severity string   `json:"severity"`
	Affected string   `json:"affected_versions"`
	URL      string   `json:"url"`
	CVEs     []string `json:"cves,omitempty"`
}

type VulnReport struct {
	Package         string          `json:"package"`
	Version         string          `json:"version"`
	Vulnerable      bool            `json:"vulnerable"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// CheckVulnerabilities queries OSV.dev for known vulnerabilities in a specific package version.
func CheckVulnerabilities(ecosystem, pkg, version string) (*VulnReport, error) {
	osvEcosystem := mapEcosystem(ecosystem)

	payload := map[string]any{
		"version": version,
		"package": map[string]string{
			"name":      pkg,
			"ecosystem": osvEcosystem,
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := httpClient.Post("https://api.osv.dev/v1/query", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("osv.dev request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("osv.dev returned %d", resp.StatusCode)
	}

	var result struct {
		Vulns []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Aliases []string `json:"aliases"`
			References []struct {
				URL string `json:"url"`
			} `json:"references"`
			DatabaseSpecific struct {
				Severity string `json:"severity"`
			} `json:"database_specific"`
		} `json:"vulns"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing osv.dev response: %w", err)
	}

	report := &VulnReport{
		Package:    pkg,
		Version:    version,
		Vulnerable: len(result.Vulns) > 0,
	}

	for _, v := range result.Vulns {
		vuln := Vulnerability{
			ID:      v.ID,
			Summary: v.Summary,
		}
		for _, alias := range v.Aliases {
			if len(alias) > 4 && alias[:4] == "CVE-" {
				vuln.CVEs = append(vuln.CVEs, alias)
			}
		}
		if len(v.References) > 0 {
			vuln.URL = v.References[0].URL
		}
		report.Vulnerabilities = append(report.Vulnerabilities, vuln)
	}

	return report, nil
}

func mapEcosystem(lang string) string {
	switch lang {
	case "npm":
		return "npm"
	case "pip":
		return "PyPI"
	case "go":
		return "Go"
	default:
		return lang
	}
}
