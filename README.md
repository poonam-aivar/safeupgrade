# SafeUpgrade 🚀🔒

AI-powered dependency upgrade agent with supply chain security built-in.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/ECR_Public-safeupgrade-FF9900?logo=amazonaws)](https://gallery.ecr.aws/i3s3o0q6/safeupgrade)

## 🎯 What is SafeUpgrade?

SafeUpgrade analyzes dependency upgrades using AI and real security data to automatically upgrade your dependencies safely. Unlike traditional tools that just check for updates, SafeUpgrade:

- 🤖 **AI-Powered Analysis** — Uses any LLM (Claude, GPT, Ollama) to read changelogs and detect breaking changes
- 🔒 **Supply Chain Security** — Checks CVEs, package provenance, and detects malicious patterns
- 📋 **Policy Enforcement** — Organization-wide upgrade policies with exceptions
- 🔄 **Automated PRs** — Creates pull requests with detailed upgrade reports
- 🎯 **Multi-Ecosystem** — Supports npm, pip, and Go
- 📦 **No Install Needed** — Parses manifest files directly (no `npm install` or `pip install` required)

## 🚀 Quick Start

### Docker (Recommended)

```bash
# Scan for outdated dependencies (no AI key needed)
docker run --rm -v $(pwd):/workspace \
  public.ecr.aws/i3s3o0q6/safeupgrade:latest scan --repo /workspace --lang pip

# Full AI-powered upgrade analysis
docker run --rm --user "$(id -u):$(id -g)" -v $(pwd):/workspace \
  -e SAFEUPGRADE_AI_KEY=your-api-key \
  -e SAFEUPGRADE_AI_URL=https://api.anthropic.com \
  public.ecr.aws/i3s3o0q6/safeupgrade:latest upgrade --repo /workspace --lang npm --policy /etc/safeupgrade/policy.yaml
```

### Install CLI

```bash
# Using Go
go install github.com/poonam-aivar/safeupgrade@latest

# Or build from source
git clone https://github.com/poonam-aivar/safeupgrade.git
cd safeupgrade && go build -o safeupgrade .
```

## 📖 Usage

### Scan for Outdated Dependencies

```bash
safeupgrade scan --repo /path/to/project --lang pip
```

```
🔍 Scanning dependencies...
  Ecosystem: pip
  Repo: /path/to/project

📦 Found 53 dependencies, 45 outdated

  ⚠️  OUTDATED fastapi: 0.128.0 → 0.137.2
  ⚠️  OUTDATED requests: 2.31 → 2.34.2
  🚨 VULNERABLE axios: 1.6.0 → 1.18.0 (CVE-2026-42044)
```

### Full Upgrade with AI Analysis

```bash
safeupgrade upgrade --repo . --lang pip --policy policy.yaml
```

```
🚀 Running SafeUpgrade agent...
  [1/5] Scanning dependencies...
        Found 35 outdated out of 35 total
  [2/5] Evaluating policies...
        27 candidates after policy filter
        27 candidates after live CVE check
  [3/5] AI analyzing changelogs and risk...
        Fetching changelogs...
        Checking CVEs...
        Verifying provenance...
        26 upgrades deemed safe
        ✅ axios: SAFE (95%) — Minor version with security hardening, verified provenance, no CVEs
        ✅ react-dom: SAFE (95%) — Patch fix, verified provenance, React core team
        ⚠️  react-hook-form: RISKY (60%) — Published <24hrs ago without provenance attestation
  [4/5] Executing upgrades...
  [5/5] Generating report...

✅ Upgrade complete: 26 upgraded, 1 skipped
   Report: upgrade_report.json
```

### Policy Check

```bash
safeupgrade policy-check --repo . --lang npm --policy policy.yaml
```

## 🤖 AI Configuration

SafeUpgrade works with **any OpenAI-compatible API**. Set two environment variables:

| Variable | What it is |
|----------|-----------|
| `SAFEUPGRADE_AI_KEY` | Your API key |
| `SAFEUPGRADE_AI_URL` | Your LLM endpoint URL |

**Examples:**

```bash
# Anthropic
export SAFEUPGRADE_AI_URL=https://api.anthropic.com
export SAFEUPGRADE_AI_KEY=sk-ant-...

# OpenAI
export SAFEUPGRADE_AI_URL=https://api.openai.com
export SAFEUPGRADE_AI_KEY=sk-...

# Azure OpenAI
export SAFEUPGRADE_AI_URL=https://your-resource.openai.azure.com
export SAFEUPGRADE_AI_KEY=your-azure-key

# Ollama (free, local, no key needed)
export SAFEUPGRADE_AI_URL=http://localhost:11434
export SAFEUPGRADE_AI_KEY=unused

# Company internal gateway
export SAFEUPGRADE_AI_URL=https://ai.yourcompany.com
export SAFEUPGRADE_AI_KEY=your-internal-key
```

> **Note:** The `scan` and `policy-check` commands work without any AI key. Only the `upgrade` command (which does changelog analysis) needs AI access.

## 🔄 CI/CD Integration

### GitHub Actions

```yaml
name: SafeUpgrade
on:
  schedule:
    - cron: '0 6 * * 1'  # Weekly on Monday
  workflow_dispatch:

jobs:
  upgrade:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - uses: actions/checkout@v4

      - name: Run SafeUpgrade
        run: |
          docker run --rm --user "$(id -u):$(id -g)" -v $(pwd):/workspace \
            -e SAFEUPGRADE_AI_KEY=${{ secrets.SAFEUPGRADE_AI_KEY }} \
            -e SAFEUPGRADE_AI_URL=${{ secrets.SAFEUPGRADE_AI_URL }} \
            public.ecr.aws/i3s3o0q6/safeupgrade:latest upgrade --repo /workspace --lang pip --policy /etc/safeupgrade/policy.yaml

      - name: Create PR
        run: |
          if [ -n "$(git diff --name-only)" ]; then
            BRANCH="safeupgrade/$(date +%Y%m%d)"
            git config user.name "SafeUpgrade Bot"
            git config user.email "safeupgrade@users.noreply.github.com"
            git checkout -b "$BRANCH"
            git add .
            git commit -m "chore(deps): safe dependency upgrades by SafeUpgrade Agent"
            git push -u origin "$BRANCH"
            PR_BODY=$(python3 -c "import sys,json; print(json.load(sys.stdin)['pr_body'])" < upgrade_report.json)
            gh pr create \
              --title "chore(deps): SafeUpgrade automated dependency update" \
              --body "$PR_BODY" \
              --base main
          else
            echo "✅ No upgrades needed"
          fi
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### GitLab CI

```yaml
safeupgrade:
  image: public.ecr.aws/i3s3o0q6/safeupgrade:latest
  script:
    - safeupgrade upgrade --repo . --lang pip --policy /etc/safeupgrade/policy.yaml
  variables:
    SAFEUPGRADE_AI_KEY: $SAFEUPGRADE_AI_KEY
    SAFEUPGRADE_AI_URL: $SAFEUPGRADE_AI_URL
  only:
    - schedules
```

### Jenkins

```groovy
pipeline {
  agent any
  stages {
    stage('SafeUpgrade') {
      steps {
        sh '''
          docker run --rm --user "$(id -u):$(id -g)" -v $WORKSPACE:/workspace \
            -e SAFEUPGRADE_AI_KEY=$SAFEUPGRADE_AI_KEY \
            -e SAFEUPGRADE_AI_URL=$SAFEUPGRADE_AI_URL \
            public.ecr.aws/i3s3o0q6/safeupgrade:latest \
            upgrade --repo /workspace --lang pip
        '''
      }
    }
  }
}
```

## 🔒 Security Features

### CVE Detection

Queries [OSV.dev](https://osv.dev) in real-time for every target version before upgrading.

**Where does vulnerability data come from?**

```
Security researcher discovers vulnerability
        ↓
Reports to GitHub Advisory Database / NVD / package registry
        ↓
OSV.dev aggregates from all sources (GitHub, NVD, PyPI, npm, Go)
        ↓
SafeUpgrade queries OSV.dev API at scan time
        ↓
Blocks upgrade if target version is affected
```

This means SafeUpgrade catches vulnerabilities within hours of public disclosure — no manual updates needed.

```
🚨 BLOCKED axios@1.14.1 — 16 known vulnerabilities
  • MAL-2026-2307: Malicious code in axios
  • CVE-2026-42044: Response tampering via prototype pollution
  • CVE-2026-40175: Unrestricted cloud metadata exfiltration
```

### Supply Chain Anomaly Detection

- 🔍 Install script detection (common attack vector)
- 🔍 Missing provenance attestation
- 🔍 Suspicious version burst publishing
- 🔍 Package publish timing anomalies

### Provenance Verification

```
✅ react@19.0.0 — Provenance verified, no install scripts, Facebook maintainer
⚠️  some-package@1.0.0 — No provenance + published <24hrs ago = RISKY
❌ malicious-pkg@1.0.0 — No provenance + install scripts = BLOCKED
```

## 🎨 Policy File

A policy file lets you define **org-wide rules** for dependency upgrades. It's optional — without it, SafeUpgrade uses sensible defaults (block canary/alpha, block major jumps).

A default policy is included in the Docker image at `/etc/safeupgrade/policy.yaml`. You can mount your own:

```bash
docker run --rm -v $(pwd):/workspace -v ./my-policy.yaml:/etc/safeupgrade/policy.yaml \
  public.ecr.aws/i3s3o0q6/safeupgrade:latest upgrade --repo /workspace --lang pip --policy /etc/safeupgrade/policy.yaml
```

Example `policy.yaml`:

```yaml
global:
  block_canary: true
  block_alpha: true
  max_major_jump: 0
  provenance_required: true

packages:
  axios:
    block_versions: ["1.14.1", "0.30.4"]
    reason: "Supply chain attack - March 2026"
  react:
    pin_major: 18
    reason: "React 19 migration planned for Q3"
  "@tanstack/router":
    block_versions: ["1.161.9", "1.161.12"]
    reason: "Supply chain attack - May 2026"
```

## 🏗️ How It's Different from Dependabot

| | Dependabot | SafeUpgrade |
|---|---|---|
| Detection | ✅ | ✅ |
| AI changelog analysis | ❌ | ✅ LLM reads actual release notes |
| Supply chain detection | ❌ | ✅ Provenance, install scripts, publish anomalies |
| Risk scoring | ❌ | ✅ Confidence-based decisions with reasoning |
| Org-wide policy | ❌ | ✅ Central YAML (block versions, pin majors) |
| Batched PRs | ❌ (1 per dep) | ✅ One PR with all safe upgrades |
| Live CVE blocking | ❌ | ✅ Blocks upgrade if target version has CVEs |
| No install needed | ✅ | ✅ |

## 📦 Supported Ecosystems

| Ecosystem | Manifest Files | Registry |
|-----------|---------------|----------|
| Python | `pyproject.toml`, `requirements.txt` | PyPI |
| Node.js | `package.json` | npm |
| Go | `go.mod` | Go proxy |

Monorepos supported — recursively scans all subdirectories.

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
git clone https://github.com/poonam-aivar/safeupgrade.git
cd safeupgrade
go mod download
go test ./...
go build -o safeupgrade .

# Test on sample project
./safeupgrade scan --repo testdata/sample-project --lang npm
```

## 📝 License

MIT License

## 🙋 Support

- 🐛 [Issue Tracker](https://github.com/poonam-aivar/safeupgrade/issues)
- 💬 [Discussions](https://github.com/poonam-aivar/safeupgrade/discussions)

---

**Made with ❤️** • [ECR Public](https://gallery.ecr.aws/i3s3o0q6/safeupgrade)
