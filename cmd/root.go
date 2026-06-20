package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "safeupgrade",
	Short: "AI-powered dependency upgrade agent with supply chain security",
}

var (
	repoPath     string
	policyFile   string
	language     string
	awsProfile   string
	awsRegion    string
	githubToken  string
	slackWebhook string
	githubOrg    string
	gatewayURL   string
	gatewayKey   string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "r", ".", "path to the repository")
	rootCmd.PersistentFlags().StringVarP(&policyFile, "policy", "p", "", "path to policy YAML file")
	rootCmd.PersistentFlags().StringVarP(&language, "lang", "l", "", "language ecosystem (npm, pip, go)")
	rootCmd.PersistentFlags().StringVar(&awsProfile, "aws-profile", "", "AWS profile for Bedrock access")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "aws-region", "us-east-1", "AWS region for Bedrock")
	rootCmd.PersistentFlags().StringVar(&githubToken, "github-token", "", "GitHub token for PR creation (or GITHUB_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL for alerts (or SLACK_WEBHOOK env)")
	rootCmd.PersistentFlags().StringVar(&githubOrg, "org", "", "GitHub org for multi-repo scanning")
	rootCmd.PersistentFlags().StringVar(&gatewayURL, "ai-url", "", "AI endpoint URL (or SAFEUPGRADE_AI_URL env)")
	rootCmd.PersistentFlags().StringVar(&gatewayKey, "ai-key", "", "AI API key (or SAFEUPGRADE_AI_KEY env)")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(policyCheckCmd)
	rootCmd.AddCommand(scanOrgCmd)
}

var scanOrgCmd = &cobra.Command{
	Use:   "scan-org",
	Short: "Scan all repositories in a GitHub organization",
	RunE:  runScanOrg,
}

func runScanOrg(cmd *cobra.Command, args []string) error {
	fmt.Println("🏢 Scanning organization repositories...")
	return executeScanOrg()
}

func Execute() error {
	return rootCmd.Execute()
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan repository for outdated and vulnerable dependencies",
	RunE:  runScan,
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade dependencies with AI analysis and policy enforcement",
	RunE:  runUpgrade,
}

var policyCheckCmd = &cobra.Command{
	Use:   "policy-check",
	Short: "Validate current dependencies against org-wide policies",
	RunE:  runPolicyCheck,
}

func runScan(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Scanning dependencies...")
	return executeScan()
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Running SafeUpgrade agent...")
	return executeUpgrade()
}

func runPolicyCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("📋 Checking policy compliance...")
	return executePolicyCheck()
}
