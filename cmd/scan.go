package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aivar-tech/safeupgrade-agent/internal/policy"
	"github.com/aivar-tech/safeupgrade-agent/internal/scanner"
)

func executeScan() error {
	lang := detectLanguage(repoPath, language)
	if lang == "" {
		return fmt.Errorf("could not detect language ecosystem. Use --lang flag (npm, pip, go)")
	}

	fmt.Printf("  Ecosystem: %s\n  Repo: %s\n\n", lang, repoPath)

	s, err := scanner.New(lang, repoPath)
	if err != nil {
		return err
	}

	report, err := s.Scan()
	if err != nil {
		return err
	}

	fmt.Printf("📦 Found %d dependencies, %d outdated\n\n", report.Total, report.Outdated)

	for _, dep := range report.Dependencies {
		if dep.Outdated {
			status := "⚠️  OUTDATED"
			if dep.Vulnerable {
				status = "🚨 VULNERABLE"
			}
			fmt.Printf("  %s %s: %s → %s\n", status, dep.Name, dep.Current, dep.Latest)
		}
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	return os.WriteFile("scan_report.json", out, 0644)
}

func executePolicyCheck() error {
	if policyFile == "" {
		return fmt.Errorf("--policy flag is required for policy-check")
	}

	pol, err := policy.Load(policyFile)
	if err != nil {
		return fmt.Errorf("loading policy: %w", err)
	}

	lang := detectLanguage(repoPath, language)
	if lang == "" {
		return fmt.Errorf("could not detect language ecosystem")
	}

	s, err := scanner.New(lang, repoPath)
	if err != nil {
		return err
	}

	report, err := s.Scan()
	if err != nil {
		return err
	}

	violations := pol.Check(report.Dependencies)
	if len(violations) == 0 {
		fmt.Println("✅ All dependencies comply with policy")
		return nil
	}

	fmt.Printf("❌ %d policy violations found:\n\n", len(violations))
	for _, v := range violations {
		fmt.Printf("  • %s: %s\n", v.Package, v.Reason)
	}
	return fmt.Errorf("%d policy violations", len(violations))
}

func detectLanguage(repo, override string) string {
	if override != "" {
		return override
	}
	if _, err := os.Stat(repo + "/package.json"); err == nil {
		return "npm"
	}
	if _, err := os.Stat(repo + "/requirements.txt"); err == nil {
		return "pip"
	}
	if _, err := os.Stat(repo + "/Pipfile"); err == nil {
		return "pip"
	}
	if _, err := os.Stat(repo + "/go.mod"); err == nil {
		return "go"
	}
	return ""
}
