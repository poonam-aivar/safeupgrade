package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AIBackend AIBackendConfig `yaml:"ai_backend"`
	GitHub    GitHubConfig    `yaml:"github"`
	Policy    PolicyConfig    `yaml:"policy"`
	Scan      ScanConfig      `yaml:"scan"`
	Upgrade   UpgradeConfig   `yaml:"upgrade"`
	Security  SecurityConfig  `yaml:"security"`
	Reporting ReportingConfig `yaml:"reporting"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Cache     CacheConfig     `yaml:"cache"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type AIBackendConfig struct {
	Provider   string `yaml:"provider"`   // openai, anthropic, bedrock, azure, ollama, gateway
	APIKey     string `yaml:"api_key"`    // Supports ${ENV_VAR} expansion
	Model      string `yaml:"model"`
	Endpoint   string `yaml:"endpoint"`   // For custom endpoints
	AWSRegion  string `yaml:"aws_region"` // For Bedrock
	AWSProfile string `yaml:"aws_profile"`
}

type GitHubConfig struct {
	Token            string   `yaml:"token"`
	EnterpriseURL    string   `yaml:"enterprise_url"`
	APIURL           string   `yaml:"api_url"`
	AutoPR           bool     `yaml:"auto_pr"`
	PRTitle          string   `yaml:"pr_title"`
	PRBaseBranch     string   `yaml:"pr_base_branch"`
	PRLabels         []string `yaml:"pr_labels"`
	PRReviewers      []string `yaml:"pr_reviewers"`
	PRTeamReviewers  []string `yaml:"pr_team_reviewers"`
}

type PolicyConfig struct {
	File   string `yaml:"file"`
	Strict bool   `yaml:"strict"`
}

type ScanConfig struct {
	Ecosystem        string `yaml:"ecosystem"`
	IncludeDev       bool   `yaml:"include_dev"`
	ConcurrentChecks int    `yaml:"concurrent_checks"`
}

type UpgradeConfig struct {
	Strategy           string `yaml:"strategy"` // all, safe, manual
	RunTests           bool   `yaml:"run_tests"`
	TestCommand        string `yaml:"test_command"`
	RollbackOnFailure  bool   `yaml:"rollback_on_failure"`
	SeparatePRs        bool   `yaml:"separate_prs"`
}

type SecurityConfig struct {
	CheckCVEs        bool `yaml:"check_cves"`
	CheckProvenance  bool `yaml:"check_provenance"`
	DetectAnomalies  bool `yaml:"detect_anomalies"`
	BlockVulnerable  bool `yaml:"block_vulnerable"`
}

type ReportingConfig struct {
	Formats      []string `yaml:"formats"` // json, markdown, sarif, html
	OutputDir    string   `yaml:"output_dir"`
	SlackWebhook string   `yaml:"slack_webhook"`
	TeamsWebhook string   `yaml:"teams_webhook"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	MaxRequestsPerHour int  `yaml:"max_requests_per_hour"`
}

type CacheConfig struct {
	Enabled   bool   `yaml:"enabled"`
	TTLHours  int    `yaml:"ttl_hours"`
	Directory string `yaml:"directory"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // text, json
	File   string `yaml:"file"`   // empty = stdout
}

// Default returns a config with sensible defaults
func Default() *Config {
	return &Config{
		AIBackend: AIBackendConfig{
			Provider: "openai",
			Model:    "gpt-4o",
		},
		GitHub: GitHubConfig{
			AutoPR:       true,
			PRTitle:      "chore(deps): SafeUpgrade automated dependency update",
			PRBaseBranch: "main",
			PRLabels:     []string{"dependencies", "safeupgrade"},
		},
		Policy: PolicyConfig{
			File:   "configs/policy.yaml",
			Strict: true,
		},
		Scan: ScanConfig{
			Ecosystem:        "auto",
			IncludeDev:       false,
			ConcurrentChecks: 5,
		},
		Upgrade: UpgradeConfig{
			Strategy:          "safe",
			RunTests:          true,
			RollbackOnFailure: true,
			SeparatePRs:       false,
		},
		Security: SecurityConfig{
			CheckCVEs:       true,
			CheckProvenance: true,
			DetectAnomalies: true,
			BlockVulnerable: true,
		},
		Reporting: ReportingConfig{
			Formats:   []string{"json", "markdown"},
			OutputDir: ".safeupgrade",
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			MaxRequestsPerHour: 10,
		},
		Cache: CacheConfig{
			Enabled:   true,
			TTLHours:  24,
			Directory: ".safeupgrade/cache",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// Load reads config from file and merges with defaults
func Load(path string) (*Config, error) {
	cfg := Default()

	// If no file specified, try default locations
	if path == "" {
		for _, p := range []string{".safeupgrade.yaml", ".safeupgrade.yml"} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	// If still no file, return defaults
	if path == "" {
		return cfg, nil
	}

	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path is user-provided CLI flag, not untrusted input
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parsing config YAML: %w", err)
	}

	// Validate and apply environment variable overrides
	cfg.applyEnvOverrides()

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	// AI Backend
	if key := os.Getenv("OPENAI_API_KEY"); key != "" && c.AIBackend.Provider == "openai" {
		c.AIBackend.APIKey = key
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" && c.AIBackend.Provider == "anthropic" {
		c.AIBackend.APIKey = key
	}
	if key := os.Getenv("AI_GATEWAY_KEY"); key != "" && c.AIBackend.Provider == "gateway" {
		c.AIBackend.APIKey = key
	}
	if url := os.Getenv("AI_GATEWAY_URL"); url != "" && c.AIBackend.Provider == "gateway" {
		c.AIBackend.Endpoint = url
	}

	// GitHub
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		c.GitHub.Token = token
	}
	if url := os.Getenv("GITHUB_ENTERPRISE_URL"); url != "" {
		c.GitHub.EnterpriseURL = url
	}

	// Reporting
	if webhook := os.Getenv("SLACK_WEBHOOK_URL"); webhook != "" {
		c.Reporting.SlackWebhook = webhook
	}
	if webhook := os.Getenv("TEAMS_WEBHOOK_URL"); webhook != "" {
		c.Reporting.TeamsWebhook = webhook
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check AI backend
	validProviders := []string{"openai", "anthropic", "bedrock", "azure", "ollama", "gateway"}
	if !contains(validProviders, c.AIBackend.Provider) {
		return fmt.Errorf("invalid AI provider: %s (must be one of: %s)",
			c.AIBackend.Provider, strings.Join(validProviders, ", "))
	}

	// Check if API key is set (except for ollama which doesn't need it)
	if c.AIBackend.Provider != "ollama" && c.AIBackend.APIKey == "" {
		return fmt.Errorf("AI API key not set for provider %s (set via config or environment variable)", c.AIBackend.Provider)
	}

	// Check upgrade strategy
	validStrategies := []string{"all", "safe", "manual"}
	if !contains(validStrategies, c.Upgrade.Strategy) {
		return fmt.Errorf("invalid upgrade strategy: %s", c.Upgrade.Strategy)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
