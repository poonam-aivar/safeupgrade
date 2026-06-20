# SafeUpgrade

AI-powered dependency upgrade agent with supply chain security.

SafeUpgrade scans your project's dependencies, checks for vulnerabilities and supply chain risks, uses AI to analyze changelogs for breaking changes, and only upgrades what's genuinely safe — then creates a PR with a full risk report.

## Quick Start

```bash
docker pull pawarpoonam/safeupgrade:latest

# Scan a project
docker run --rm -v $(pwd):/workspace \
  pawarpoonam/safeupgrade:latest scan --repo /workspace --lang pip
```

## Usage

### Scan (detect outdated dependencies)

```bash
# Python project
docker run --rm -v $(pwd):/workspace \
  pawarpoonam/safeupgrade:latest scan --repo /workspace --lang pip

# Node.js project
docker run --rm -v $(pwd):/workspace \
  pawarpoonam/safeupgrade:latest scan --repo /workspace --lang npm

# Go project
docker run --rm -v $(pwd):/workspace \
  pawarpoonam/safeupgrade:latest scan --repo /workspace --lang go
```

No installation needed — it parses `package.json`, `pyproject.toml`, and `requirements.txt` directly.

### Full Upgrade (AI analysis + file changes)

```bash
docker run --rm --user root -v $(pwd):/workspace \
  -e SAFEUPGRADE_AI_KEY=your-llm-api-key \
  pawarpoonam/safeupgrade:latest upgrade --repo /workspace --lang pip --policy /etc/safeupgrade/policy.yaml
```

This will:
1. Scan all dependency files in your project
2. Check each target version for known CVEs (via OSV.dev)
3. Send real changelog + provenance data to an LLM (Claude) for risk analysis
4. Update version pins in your source files for packages deemed SAFE
5. Generate `upgrade_report.json` with a full risk report

### Policy Check

```bash
docker run --rm -v $(pwd):/workspace \
  pawarpoonam/safeupgrade:latest policy-check --repo /workspace --lang npm --policy /etc/safeupgrade/policy.yaml
```

## GitHub Actions

Add this workflow to any repo you want SafeUpgrade to protect:

```yaml
name: SafeUpgrade

on:
  schedule:
    - cron: '0 6 * * 1'  # Every Monday 6 AM UTC
  workflow_dispatch:       # Or trigger manually

jobs:
  safeupgrade:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - uses: actions/checkout@v4

      - name: Run SafeUpgrade
        run: |
          docker run --rm --user root -v $(pwd):/workspace \
            -e SAFEUPGRADE_AI_KEY=${{ secrets.SAFEUPGRADE_AI_KEY }} \
            pawarpoonam/safeupgrade:latest upgrade --repo /workspace --lang pip --policy /etc/safeupgrade/policy.yaml

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

> **Why is PR creation a separate step?**
> SafeUpgrade runs inside Docker — it analyzes deps and updates version pins in your files. But creating a PR requires git push + GitHub API access, which is handled by the runner (not the container). This is the same pattern used by tools like Renovate and Terraform.

### Setup

Add one secret to your repo (Settings → Secrets → Actions):

| Secret | What it is |
|--------|-----------|
| `SAFEUPGRADE_AI_KEY` | API key for the LLM service that powers the AI analysis (e.g., your org's AI gateway, or an Anthropic/OpenAI key). This is **not** an AWS API Gateway key — it's the authentication token for the AI model endpoint that SafeUpgrade calls to analyze changelogs and assess risk. |

`GITHUB_TOKEN` is provided automatically — no setup needed.

Change `--lang pip` to `npm` or `go` depending on your project. For monorepos, point `--repo` at the root — it recursively finds all dependency files.

## What It Does

```
[1/5] Scan         → Finds outdated deps (parses manifests directly — no install needed)
[2/5] Policy       → Blocks known-compromised versions + major jumps
      Live CVE     → Queries OSV.dev for vulnerabilities in each target version
[3/5] AI Analysis  → LLM reads changelogs, checks provenance, scores risk → SAFE / RISKY / BLOCK
[4/5] Execute      → Updates version pins in source files (only SAFE packages)
[5/5] Report       → Generates upgrade_report.json with PR-ready markdown
```

## Example Output

```
🚀 Running SafeUpgrade agent...
  [1/5] Scanning dependencies...
        Found 35 outdated out of 35 total
  [2/5] Evaluating policies...
        27 candidates after policy filter
        27 candidates after live CVE check
  [3/5] AI analyzing changelogs and risk...
        26 upgrades deemed safe
        ✅ axios: SAFE (95%) — Minor version with security hardening, verified provenance, no CVEs
        ✅ react-dom: SAFE (95%) — Patch fix, verified provenance, React core team
        ⚠️  react-hook-form: RISKY (60%) — Published <24hrs ago without provenance attestation
  [4/5] Executing upgrades...
  [5/5] Generating report...

✅ Upgrade complete: 26 upgraded, 1 skipped
```

## How It's Different from Dependabot

| | Dependabot | SafeUpgrade |
|---|---|---|
| Detection | ✅ | ✅ |
| AI changelog analysis | ❌ | ✅ LLM reads actual release notes |
| Supply chain detection | ❌ | ✅ Provenance, install scripts, publish anomalies |
| Risk scoring | ❌ | ✅ Confidence-based decisions with reasoning |
| Org-wide policy | ❌ | ✅ Central YAML (block versions, pin majors) |
| Batched PRs | ❌ (1 per dep) | ✅ One PR with all safe upgrades |
| Live CVE blocking | ❌ | ✅ Blocks upgrade if target version has CVEs |

## Supported Ecosystems

| Ecosystem | Manifest Files | Registry |
|-----------|---------------|----------|
| Python | `pyproject.toml`, `requirements.txt` | PyPI |
| Node.js | `package.json` | npm |
| Go | `go.mod` | Go proxy |

Monorepos supported — recursively scans all subdirectories.

## Policy File

Enforce org-wide rules:

```yaml
global:
  block_canary: true
  block_alpha: true
  max_major_jump: 0

packages:
  axios:
    block_versions: ["1.14.1", "0.30.4"]
    reason: "Supply chain attack - March 2026"
  react:
    pin_major: 18
    reason: "React 19 migration planned for Q3"
```

## Docker

```bash
docker pull pawarpoonam/safeupgrade:latest
```

[Docker Hub →](https://hub.docker.com/r/pawarpoonam/safeupgrade)

## License

MIT
