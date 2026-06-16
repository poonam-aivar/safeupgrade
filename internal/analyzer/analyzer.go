package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aivar-tech/safeupgrade-agent/internal/scanner"
	"github.com/aivar-tech/safeupgrade-agent/internal/tools"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Result struct {
	Package            string   `json:"package"`
	From               string   `json:"from"`
	To                 string   `json:"to"`
	ChangelogSummary   string   `json:"changelog_summary"`
	BreakingChanges    bool     `json:"breaking_changes_detected"`
	AnomalyScore       float64  `json:"anomaly_score"`
	Confidence         float64  `json:"confidence"`
	Recommendation     string   `json:"recommendation"`
	Reasoning          string   `json:"reasoning"`
	ProvenanceVerified bool     `json:"provenance_verified"`
	MaintainerChanged  bool     `json:"maintainer_change"`
	CVEs               []string `json:"cves,omitempty"`
}

type Analyzer struct {
	bedrockClient *bedrockruntime.Client
	gatewayURL    string
	gatewayKey    string
	ecosystem     string
}

func New() *Analyzer {
	return NewWithConfig("npm", "", "us-east-1", "", "")
}

func NewWithEcosystem(ecosystem string) *Analyzer {
	return NewWithConfig(ecosystem, "", "us-east-1", "", "")
}

func NewWithConfig(ecosystem, awsProfile, awsRegion, gatewayURL, gatewayKey string) *Analyzer {
	a := &Analyzer{ecosystem: ecosystem}

	// Priority 1: AI Gateway
	if gatewayURL == "" {
		gatewayURL = os.Getenv("AI_GATEWAY_URL")
	}
	if gatewayKey == "" {
		gatewayKey = os.Getenv("AI_GATEWAY_KEY")
	}
	if gatewayURL != "" && gatewayKey != "" {
		a.gatewayURL = gatewayURL
		a.gatewayKey = gatewayKey
		return a
	}

	// Priority 2: Bedrock
	opts := []func(*config.LoadOptions) error{}
	if awsProfile != "" {
		opts = append(opts, config.WithSharedConfigProfile(awsProfile))
	}
	if awsRegion != "" {
		opts = append(opts, config.WithRegion(awsRegion))
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err == nil {
		a.bedrockClient = bedrockruntime.NewFromConfig(cfg)
	}

	return a
}

func (a *Analyzer) Analyze(deps []scanner.Dependency) ([]Result, error) {
	if a.gatewayURL == "" && a.bedrockClient == nil {
		return nil, fmt.Errorf("no AI backend configured (set AI_GATEWAY_URL+AI_GATEWAY_KEY or AWS profile)")
	}

	// Step 1: Gather real data using tools
	fmt.Println("        Fetching changelogs...")
	changelogs := a.fetchChangelogs(deps)
	fmt.Println("        Checking CVEs...")
	vulnReports := a.checkCVEs(deps)
	fmt.Println("        Verifying provenance...")
	metadata := a.checkProvenance(deps)

	// Step 2: Run agentic loop — Claude reasons over real data
	return a.agentLoop(deps, changelogs, vulnReports, metadata)
}

func (a *Analyzer) agentLoop(
	deps []scanner.Dependency,
	changelogs map[string]*tools.Changelog,
	vulns map[string]*tools.VulnReport,
	metadata map[string]*tools.PackageMetadata,
) ([]Result, error) {

	prompt := buildAgentPrompt(deps, changelogs, vulns, metadata)
	systemPrompt := `You are a dependency upgrade safety agent. You have been given REAL data: 
actual changelogs, CVE reports, and package provenance metadata.

Your job is to analyze each dependency upgrade and decide if it's SAFE, RISKY, or BLOCK.

Rules:
- BLOCK if the target version has known CVEs
- BLOCK if provenance is missing AND install scripts are present (supply chain risk)
- BLOCK if multiple versions published in burst (< 1 hour)
- RISKY if major version jump with breaking changes in changelog
- RISKY if published < 24 hours ago with no provenance
- SAFE if minor/patch with no CVEs, provenance verified, no anomalies

Respond ONLY with a JSON array. No markdown, no explanation outside the JSON.`

	var responseText string
	var err error

	if a.gatewayURL != "" {
		responseText, err = a.callGateway(systemPrompt, prompt)
	} else {
		responseText, err = a.callBedrock(systemPrompt, prompt)
	}

	if err != nil {
		return nil, err
	}

	return parseResponse(responseText, deps, metadata)
}

// callGateway calls the AI Gateway (OpenAI-compatible API)
func (a *Analyzer) callGateway(system, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": "claude-sonnet-4.5",
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 4096,
	})

	url := strings.TrimRight(a.gatewayURL, "/") + "/v1/chat/completions"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.gatewayKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gateway request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing gateway response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("gateway returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}

// callBedrock calls AWS Bedrock directly
func (a *Analyzer) callBedrock(system, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        4096,
		"system":            system,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	resp, err := a.bedrockClient.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		ModelId:     strPtr("anthropic.claude-sonnet-4-20250514-v1:0"),
		ContentType: strPtr("application/json"),
		Body:        body,
	})
	if err != nil {
		return "", fmt.Errorf("bedrock invoke: %w", err)
	}

	var bedrockResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Body, &bedrockResp); err != nil {
		return "", err
	}
	if len(bedrockResp.Content) == 0 {
		return "", fmt.Errorf("bedrock returned no content")
	}
	return bedrockResp.Content[0].Text, nil
}

func buildAgentPrompt(
	deps []scanner.Dependency,
	changelogs map[string]*tools.Changelog,
	vulns map[string]*tools.VulnReport,
	metadata map[string]*tools.PackageMetadata,
) string {
	var sb strings.Builder
	sb.WriteString("Analyze these dependency upgrades using the real data provided:\n\n")

	for _, dep := range deps {
		sb.WriteString(fmt.Sprintf("## %s: %s → %s\n", dep.Name, dep.Current, dep.Latest))

		if cl, ok := changelogs[dep.Name]; ok && cl.Body != "" {
			sb.WriteString(fmt.Sprintf("### Changelog (from %s):\n%s\n", cl.Source, cl.Body))
		} else {
			sb.WriteString("### Changelog: NOT AVAILABLE\n")
		}

		if vr, ok := vulns[dep.Name]; ok {
			if vr.Vulnerable {
				sb.WriteString("### ⚠️ VULNERABILITIES in target version:\n")
				for _, v := range vr.Vulnerabilities {
					sb.WriteString(fmt.Sprintf("- %s: %s (CVEs: %v)\n", v.ID, v.Summary, v.CVEs))
				}
			} else {
				sb.WriteString("### CVEs: None found in target version ✓\n")
			}
		}

		if meta, ok := metadata[dep.Name]; ok {
			sb.WriteString("### Package Metadata:\n")
			sb.WriteString(fmt.Sprintf("- Provenance: %v\n", meta.HasProvenance))
			sb.WriteString(fmt.Sprintf("- Install scripts: %v\n", meta.HasInstallScripts))
			sb.WriteString(fmt.Sprintf("- Published: %s\n", meta.PublishedAt.Format("2006-01-02 15:04 UTC")))
			sb.WriteString(fmt.Sprintf("- Maintainers: %v\n", meta.Maintainers))
			if len(meta.Anomalies) > 0 {
				sb.WriteString(fmt.Sprintf("- 🚨 ANOMALIES: %v\n", meta.Anomalies))
			}
		}
		sb.WriteString("\n---\n\n")
	}

	sb.WriteString(`
For each package, respond with this JSON structure:
[
  {
    "package": "name",
    "from": "current",
    "to": "target",
    "changelog_summary": "brief summary of changes",
    "breaking_changes_detected": true/false,
    "anomaly_score": 0.0-1.0,
    "confidence": 0.0-1.0,
    "recommendation": "SAFE|RISKY|BLOCK",
    "reasoning": "one sentence explanation"
  }
]`)
	return sb.String()
}

func parseResponse(text string, deps []scanner.Dependency, metadata map[string]*tools.PackageMetadata) ([]Result, error) {
	// Strip markdown code blocks if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 {
		return FallbackAnalysis(deps), nil
	}

	var results []Result
	if err := json.Unmarshal([]byte(text[start:end+1]), &results); err != nil {
		return FallbackAnalysis(deps), nil
	}

	for i, r := range results {
		if meta, ok := metadata[r.Package]; ok {
			results[i].ProvenanceVerified = meta.HasProvenance
		}
	}

	return results, nil
}

func (a *Analyzer) fetchChangelogs(deps []scanner.Dependency) map[string]*tools.Changelog {
	results := make(map[string]*tools.Changelog)
	for _, dep := range deps {
		cl, err := tools.FetchChangelog(dep.Name, dep.Latest)
		if err == nil {
			results[dep.Name] = cl
		}
	}
	return results
}

func (a *Analyzer) checkCVEs(deps []scanner.Dependency) map[string]*tools.VulnReport {
	results := make(map[string]*tools.VulnReport)
	for _, dep := range deps {
		vr, err := tools.CheckVulnerabilities(a.ecosystem, dep.Name, dep.Latest)
		if err == nil {
			results[dep.Name] = vr
		}
	}
	return results
}

func (a *Analyzer) checkProvenance(deps []scanner.Dependency) map[string]*tools.PackageMetadata {
	results := make(map[string]*tools.PackageMetadata)
	if a.ecosystem != "npm" {
		return results
	}
	for _, dep := range deps {
		meta, err := tools.CheckPackageMetadata(dep.Name, dep.Latest)
		if err == nil {
			results[dep.Name] = meta
		}
	}
	return results
}

// FallbackAnalysis uses rule-based heuristics when AI is unavailable.
func FallbackAnalysis(deps []scanner.Dependency) []Result {
	results := make([]Result, 0, len(deps))
	for _, dep := range deps {
		rec := "SAFE"
		confidence := 0.7

		currentMajor := getMajorVersion(dep.Current)
		latestMajor := getMajorVersion(dep.Latest)

		if currentMajor != latestMajor {
			rec = "RISKY"
			confidence = 0.4
		}
		if isPrerelease(dep.Latest) {
			rec = "BLOCK"
			confidence = 0.9
		}

		results = append(results, Result{
			Package:        dep.Name,
			From:           dep.Current,
			To:             dep.Latest,
			Recommendation: rec,
			Confidence:     confidence,
			Reasoning:      "Rule-based fallback (AI unavailable)",
		})
	}
	return results
}

func getMajorVersion(v string) string {
	v = strings.TrimLeft(v, "^~>=<v")
	parts := strings.Split(v, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return v
}

func isPrerelease(v string) bool {
	lower := strings.ToLower(v)
	return strings.Contains(lower, "alpha") ||
		strings.Contains(lower, "beta") ||
		strings.Contains(lower, "canary") ||
		strings.Contains(lower, "rc")
}

func strPtr(s string) *string { return &s }
