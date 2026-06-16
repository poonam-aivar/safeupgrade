# SafeUpgrade Agent 🚀🔒

AI-powered dependency upgrade agent with supply chain security built-in.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![Docker Pulls](https://img.shields.io/docker/pulls/aivar/safeupgrade)](https://hub.docker.com/r/aivar/safeupgrade)

## 🎯 What is SafeUpgrade?

SafeUpgrade analyzes dependency upgrades using AI and real security data to automatically upgrade your dependencies safely. Unlike traditional tools that just check for updates, SafeUpgrade:

- 🤖 **AI-Powered Analysis** - Uses Claude/GPT to read changelogs and detect breaking changes
- 🔒 **Supply Chain Security** - Checks CVEs, package provenance, and detects malicious patterns
- 📋 **Policy Enforcement** - Organization-wide upgrade policies with exceptions
- 🔄 **Automated PRs** - Creates pull requests with detailed upgrade reports
- 🎯 **Multi-Ecosystem** - Supports npm, pip, and Go (more coming)

## 🚀 Quick Start

### Option 1: Try Online (No Installation)

Visit [safeupgrade.io](https://safeupgrade.io) and paste your GitHub repo URL to get a free security analysis.

### Option 2: Use in CI/CD (Like Snyk/Trivy)

```yaml
# .github/workflows/safeupgrade.yml
name: SafeUpgrade

on:
  schedule:
    - cron: '0 6 * * 1'  # Weekly on Monday
  workflow_dispatch:

jobs:
  upgrade:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run SafeUpgrade
        uses: docker://aivar/safeupgrade:latest
        with:
          args: upgrade --auto-pr
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Option 3: Install CLI

```bash
# macOS/Linux
brew install aivar/tap/safeupgrade

# Or download binary
curl -sSL https://get.safeupgrade.io | sh

# Or use Go
go install github.com/aivar-tech/safeupgrade-agent@latest
```

### Option 4: Docker

```bash
docker run -v $(pwd):/repo \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  aivar/safeupgrade:latest \
  upgrade --repo /repo
```

## 📖 Usage

### Scan for Outdated Dependencies

```bash
safeupgrade scan
```

Output:
```
🔍 Scanning dependencies...
  Ecosystem: npm
  Repo: /Users/you/myproject

📦 Found 45 dependencies, 8 outdated

  ⚠️  OUTDATED react: 18.2.0 → 19.0.0
  ⚠️  OUTDATED axios: 1.6.0 → 1.6.5
  🚨 VULNERABLE express: 4.18.0 → 4.18.2 (CVE-2024-12345)
```

### Upgrade with AI Analysis

```bash
safeupgrade upgrade --policy configs/policy.yaml
```

The agent will:
1. ✅ Scan for outdated dependencies
2. 🔍 Check against your policy rules
3. 🔒 Verify CVEs and provenance
4. 🤖 AI analyzes changelogs for breaking changes
5. ⬆️ Upgrades safe dependencies
6. ✅ Runs your test suite
7. 📝 Creates a PR with detailed report

### Configure Your AI Backend

SafeUpgrade supports multiple AI providers:

```bash
# OpenAI (recommended for public use)
export OPENAI_API_KEY=sk-...
safeupgrade upgrade

# Anthropic
export ANTHROPIC_API_KEY=sk-ant-...
safeupgrade upgrade --ai-provider anthropic

# AWS Bedrock (for enterprise)
safeupgrade upgrade --ai-provider bedrock --aws-region us-east-1

# Ollama (free, local)
safeupgrade upgrade --ai-provider ollama --ai-endpoint http://localhost:11434

# Custom Gateway (for your company)
safeupgrade upgrade \
  --gateway-url https://ai-gateway.yourcompany.com \
  --gateway-key your-key
```

## 🎨 Configuration

Create `.safeupgrade.yaml` in your repo:

```yaml
ai_backend:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o"

github:
  token: "${GITHUB_TOKEN}"
  auto_pr: true

policy:
  file: "configs/policy.yaml"

upgrade:
  strategy: "safe"
  run_tests: true
  rollback_on_failure: true

security:
  check_cves: true
  check_provenance: true
  block_vulnerable: true
```

### Policy File

```yaml
# configs/policy.yaml
global:
  block_canary: true
  block_alpha: true
  max_major_jump: 1
  provenance_required: true

packages:
  react:
    pin_major: 18
    reason: "React 19 migration planned for Q3"
  
  axios:
    block_versions: ["1.14.1"]
    reason: "Supply chain attack detected"

alerts:
  compromised_package:
    channel: "#security-alerts"
    action: "auto-pin-previous"
```

## 🏢 Enterprise Deployment

For internal company use with your own AI models:

```bash
# Deploy to Kubernetes
helm install safeupgrade ./helm/safeupgrade \
  --set config.aiGatewayURL=https://ai.yourcompany.com \
  --set config.githubEnterpriseURL=https://github.yourcompany.com

# Or use Docker Compose
docker-compose -f docker-compose.enterprise.yml up -d
```

See [Enterprise Deployment Guide](docs/ENTERPRISE_DEPLOYMENT.md) for details.

## 🔒 Security Features

### CVE Detection
Checks OSV.dev database for known vulnerabilities in target versions:
```
🚨 BLOCKED axios@1.14.1 — 3 known vulnerabilities
   • CVE-2024-12345: Code injection vulnerability
   • CVE-2024-12346: Prototype pollution
```

### Supply Chain Anomaly Detection
- Install script detection (common attack vector)
- Missing provenance attestation
- Maintainer changes
- Suspicious version bursts
- Package age validation

### Provenance Verification
For npm packages, verifies build provenance using sigstore attestations:
```
✅ react@19.0.0 - Provenance verified
❌ some-package@1.0.0 - No provenance + install scripts = BLOCKED
```

## 📊 Output Formats

### JSON (default)
```bash
safeupgrade scan --output json > report.json
```

### SARIF (GitHub Security Tab)
```bash
safeupgrade scan --output sarif > results.sarif
# Upload to GitHub Security tab
```

### Markdown (for PRs)
```bash
safeupgrade upgrade --output markdown > report.md
```

### HTML (visual report)
```bash
safeupgrade scan --output html > report.html
```

## 🔄 CI/CD Integration Examples

<details>
<summary><b>GitHub Actions</b></summary>

```yaml
- uses: docker://aivar/safeupgrade:latest
  with:
    args: upgrade --auto-pr
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
</details>

<details>
<summary><b>GitLab CI</b></summary>

```yaml
safeupgrade:
  image: aivar/safeupgrade:latest
  script:
    - safeupgrade upgrade --auto-pr
  variables:
    OPENAI_API_KEY: $OPENAI_API_KEY
    GITHUB_TOKEN: $CI_JOB_TOKEN
  only:
    - schedules
```
</details>

<details>
<summary><b>Jenkins</b></summary>

```groovy
pipeline {
  agent any
  stages {
    stage('SafeUpgrade') {
      steps {
        sh '''
          docker run -v $WORKSPACE:/repo \
            -e OPENAI_API_KEY=$OPENAI_API_KEY \
            aivar/safeupgrade:latest \
            upgrade --repo /repo
        '''
      }
    }
  }
}
```
</details>

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repo
git clone https://github.com/aivar-tech/safeupgrade-agent
cd safeupgrade-agent

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o safeupgrade .

# Test on sample project
./safeupgrade scan --repo testdata/sample-project
```

## 📝 License

MIT License - see [LICENSE](LICENSE)

## 🙋 Support

- 📚 [Documentation](https://docs.safeupgrade.io)
- 💬 [GitHub Discussions](https://github.com/aivar-tech/safeupgrade-agent/discussions)
- 🐛 [Issue Tracker](https://github.com/aivar-tech/safeupgrade-agent/issues)
- 🔒 [Security Policy](SECURITY.md)

## 🌟 Star History

If SafeUpgrade helps you, give it a ⭐️ on GitHub!

## 🗺️ Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features.

---

**Made with ❤️ by [Aivar](https://aivar.app)** • [Website](https://safeupgrade.io) • [Blog](https://blog.aivar.app)
