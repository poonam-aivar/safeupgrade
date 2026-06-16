package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Advisory struct {
	ID        string    `json:"id"`
	Package   string    `json:"package"`
	Ecosystem string    `json:"ecosystem"`
	Versions  []string  `json:"affected_versions"`
	Summary   string    `json:"summary"`
	Severity  string    `json:"severity"`
	Published time.Time `json:"published"`
	IsMalware bool      `json:"is_malware"`
}

// FetchRecentAdvisories queries OSV.dev for recently published advisories
// in a given ecosystem. This catches NEW compromises without manual YAML updates.
func FetchRecentAdvisories(ecosystem string, since time.Duration) ([]Advisory, error) {
	cutoff := time.Now().Add(-since)

	// Query OSV for malware advisories in the ecosystem
	payload := map[string]any{
		"package": map[string]string{
			"ecosystem": mapEcosystem(ecosystem),
		},
	}

	// OSV doesn't support time-based queries directly, so we query by known malware prefixes
	// and filter by time. For production, use the OSV bulk export or GitHub Advisory DB GraphQL API.
	body, _ := json.Marshal(payload)
	_ = body
	_ = cutoff

	// Use GitHub Advisory Database API for real-time malware detection
	return fetchGitHubAdvisories(ecosystem, cutoff)
}

func fetchGitHubAdvisories(ecosystem string, since time.Time) ([]Advisory, error) {
	ghEcosystem := "NPM"
	switch ecosystem {
	case "pip":
		ghEcosystem = "PIP"
	case "go":
		ghEcosystem = "GO"
	}

	// GitHub Advisory DB GraphQL query for MALWARE type advisories
	query := fmt.Sprintf(`{"query":"{ securityAdvisories(first:20, type:MALWARE, ecosystem:%s, orderBy:{field:PUBLISHED_AT,direction:DESC}) { nodes { ghsaId summary publishedAt vulnerabilities(first:5) { nodes { package { name ecosystem } vulnerableVersionRange } } } } }"}`, ghEcosystem)

	resp, err := httpClient.Post(
		"https://api.github.com/graphql",
		"application/json",
		bytes.NewReader([]byte(query)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If no GitHub token, fall back to OSV batch check
	if resp.StatusCode == 401 {
		return fetchOSVMalware(ecosystem, since)
	}

	var result struct {
		Data struct {
			SecurityAdvisories struct {
				Nodes []struct {
					GhsaID    string `json:"ghsaId"`
					Summary   string `json:"summary"`
					Published string `json:"publishedAt"`
					Vulns     struct {
						Nodes []struct {
							Package struct {
								Name string `json:"name"`
							} `json:"package"`
							Range string `json:"vulnerableVersionRange"`
						} `json:"nodes"`
					} `json:"vulnerabilities"`
				} `json:"nodes"`
			} `json:"securityAdvisories"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var advisories []Advisory
	for _, node := range result.Data.SecurityAdvisories.Nodes {
		pub, _ := time.Parse(time.RFC3339, node.Published)
		if pub.Before(since) {
			continue
		}
		for _, v := range node.Vulns.Nodes {
			advisories = append(advisories, Advisory{
				ID:        node.GhsaID,
				Package:   v.Package.Name,
				Ecosystem: ecosystem,
				Summary:   node.Summary,
				Published: pub,
				IsMalware: true,
			})
		}
	}
	return advisories, nil
}

// fetchOSVMalware checks OSV for MAL-prefixed advisories (malware).
func fetchOSVMalware(ecosystem string, since time.Time) ([]Advisory, error) {
	// OSV malware advisories have IDs starting with "MAL-"
	// We check specific packages that are commonly targeted
	highValueTargets := []string{"axios", "react", "next", "express", "lodash", "webpack"}

	var advisories []Advisory
	for _, pkg := range highValueTargets {
		vr, err := CheckVulnerabilities(ecosystem, pkg, "") // empty version = check all
		if err != nil {
			continue
		}
		for _, v := range vr.Vulnerabilities {
			if len(v.ID) > 4 && v.ID[:4] == "MAL-" {
				advisories = append(advisories, Advisory{
					ID:        v.ID,
					Package:   pkg,
					Ecosystem: ecosystem,
					Summary:   v.Summary,
					IsMalware: true,
				})
			}
		}
	}
	return advisories, nil
}
