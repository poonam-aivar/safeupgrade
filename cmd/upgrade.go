package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aivar-tech/safeupgrade-agent/internal/analyzer"
	"github.com/aivar-tech/safeupgrade-agent/internal/executor"
	"github.com/aivar-tech/safeupgrade-agent/internal/policy"
	"github.com/aivar-tech/safeupgrade-agent/internal/reporter"
	"github.com/aivar-tech/safeupgrade-agent/internal/scanner"
	"github.com/aivar-tech/safeupgrade-agent/internal/tools"
)

func executeUpgrade() error {
	lang := detectLanguage(repoPath, language)
	if lang == "" {
		return fmt.Errorf("could not detect language ecosystem. Use --lang flag")
	}

	// Step 1: Scan
	fmt.Println("  [1/5] Scanning dependencies...")
	s, err := scanner.New(lang, repoPath)
	if err != nil {
		return err
	}
	report, err := s.Scan()
	if err != nil {
		return err
	}
	fmt.Printf("        Found %d outdated out of %d total\n", report.Outdated, report.Total)

	if report.Outdated == 0 {
		fmt.Println("\n✅ All dependencies are up to date")
		return nil
	}

	// Step 2: Policy check
	fmt.Println("  [2/5] Evaluating policies...")
	var pol *policy.Policy
	if policyFile != "" {
		pol, err = policy.Load(policyFile)
		if err != nil {
			return fmt.Errorf("loading policy: %w", err)
		}
	} else {
		pol = policy.Default()
	}

	candidates := pol.Filter(report.Dependencies)
	fmt.Printf("        %d candidates after policy filter\n", len(candidates))

	// Step 2b: Live CVE check — block any target version with known vulnerabilities
	if len(candidates) > 0 {
		var safeCandidates []scanner.Dependency
		for _, dep := range candidates {
			vr, err := tools.CheckVulnerabilities(lang, dep.Name, dep.Latest)
			if err == nil && vr.Vulnerable {
				fmt.Printf("        🚨 BLOCKED %s@%s — %d known vulnerabilities\n", dep.Name, dep.Latest, len(vr.Vulnerabilities))
				continue
			}
			safeCandidates = append(safeCandidates, dep)
		}
		candidates = safeCandidates
		fmt.Printf("        %d candidates after live CVE check\n", len(candidates))
	}

	if len(candidates) == 0 {
		fmt.Println("\n⚠️  No upgrades pass policy checks")
		return nil
	}

	// Step 3: AI analysis
	fmt.Println("  [3/5] AI analyzing changelogs and risk...")
	ai := analyzer.NewWithConfig(lang, awsProfile, awsRegion, gatewayURL, gatewayKey)
	analysis, err := ai.Analyze(candidates)
	if err != nil {
		fmt.Printf("        ⚠️  AI analysis unavailable: %v\n", err)
		fmt.Println("        Falling back to rule-based scoring...")
		analysis = analyzer.FallbackAnalysis(candidates)
	}

	safe := filterSafe(analysis)
	fmt.Printf("        %d upgrades deemed safe\n", len(safe))

	// Show AI decisions
	for _, a := range analysis {
		icon := "✅"
		if a.Recommendation == "RISKY" {
			icon = "⚠️ "
		} else if a.Recommendation == "BLOCK" {
			icon = "🚨"
		}
		fmt.Printf("        %s %s: %s (confidence: %.0f%%) — %s\n", icon, a.Package, a.Recommendation, a.Confidence*100, a.Reasoning)
	}

	if len(safe) == 0 {
		fmt.Println("\n⚠️  No upgrades deemed safe by AI analysis")
		return nil
	}

	// Step 4: Execute upgrades
	fmt.Println("  [4/5] Executing upgrades...")
	exec := executor.New(lang, repoPath)
	result, err := exec.Upgrade(safe)
	if err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Step 5: Report
	fmt.Println("  [5/5] Generating report...")
	rep := reporter.Generate(report, analysis, result)

	out, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile("upgrade_report.json", out, 0644); err != nil {
		return err
	}

	fmt.Printf("\n✅ Upgrade complete: %d upgraded, %d skipped\n", result.Upgraded, result.Skipped)
	fmt.Println("   Report: upgrade_report.json")
	return nil
}

func filterSafe(analyses []analyzer.Result) []analyzer.Result {
	var safe []analyzer.Result
	for _, a := range analyses {
		if a.Recommendation == "SAFE" {
			safe = append(safe, a)
		}
	}
	return safe
}
