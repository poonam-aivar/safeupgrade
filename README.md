# SafeUpgrade

AI-powered dependency upgrade agent with supply chain security.

SafeUpgrade scans your project's dependencies, checks for vulnerabilities and supply chain risks, uses AI to analyze changelogs for breaking changes, and only upgrades what's genuinely safe — then creates a PR with a full risk report.

## Quick Start

```bash
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

### Full Upgrade (AI analysis + file changes)

```bash
docker run --rm --user root -v $(pwd):/workspace \
  -e AI_GATEWAY_KEY=your-key \
  pawarpoonam/safeupgrade:latest upgrade --repo /workspace --lang pip --policy /etc/safeupgrade/policy.yaml
```

### Policy Check (validate against org rules)

```bash
docker run --rm -v $(pwd):/workspace \
  pawarpoonam/safeupgrade:latest policy-check --repo /workspace --lang npm --policy /etc/safeupgrade/policy.yaml
```

## GitHub Actions

Add this to any repo's workflow:

```yaml
name: SafeUpgrade

on:
  schedule:
    - cron: '0 6 * * 1'  # Every Monday 6 AM UTC
  workflow_dispatch:

jobs:
  safeupgrade:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: SafeUpgrade Full Analysis
        run: |
          docker run --rm --user root -v $(pwd):/workspace \
            -e AI_GATEWAY_KEY=${{ secrets.AI_GATEWAY_KEY }} \
            pawarpoonam/safeupgrade:latest upgrade --repo /workspace --lang pip --policy /etc/safeupgrade/policy.yaml

      - name: Create PR if upgrades were made
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
            echo "No upgrades applied — all deps either up-to-date or blocked by policy/AI"
          fi
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Required secret:** `AI_GATEWAY_KEY` — your AI gateway API key for Claude analysis.

Change `--lang pip` to `npm` or `go` depending on your project.

## What It Does

```
[1/5] Scan         → Finds outdated deps (parses package.json / pyproject.toml / requirements.txt directly — no install needed)
[2/5] Policy       → Blocks known-compromised versions + major jumps
      Live CVE     → Queries OSV.dev for each target version
[3/5] AI Analysis  → Claude reads changelogs, checks provenance, scores risk → SAFE / RISKY / BLOCK
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
        Fetching changelogs...
        Checking CVEs...
        Verifying provenance...
        26 upgrades deemed safe
        ✅ axios: SAFE (confidence: 95%) — Minor version with security hardening, verified provenance, no CVEs
        ✅ react-dom: SAFE (confidence: 95%) — Patch fix, verified provenance, React core team
        ⚠️  react-hook-form: RISKY (confidence: 60%) — Published <24 hours ago without provenance attestation
  [4/5] Executing upgrades...
  [5/5] Generating report...

✅ Upgrade complete: 26 upgraded, 1 skipped
```

## How It's Different from Dependabot

| | Dependabot | SafeUpgrade |
|---|---|---|
| Detection | ✅ | ✅ |
| AI changelog analysis | ❌ | ✅ Claude reads release notes |
| Supply chain detection | ❌ | ✅ Provenance, install scripts, publish anomalies |
| Risk scoring | ❌ | ✅ Confidence-based decisions |
| Org-wide policy | ❌ | ✅ Central YAML (block versions, pin majors) |
| Batched PRs | ❌ (1 per dep) | ✅ One PR with all safe upgrades |
| Live CVE blocking | ❌ | ✅ OSV.dev query before upgrade |
| No install needed | ✅ | ✅ Parses manifest files directly |

## Supported Ecosystems

| Ecosystem | Manifest Files | Registry |
|-----------|---------------|----------|
| Python | `pyproject.toml`, `requirements.txt` | PyPI |
| Node.js | `package.json` | npm |
| Go | `go.mod` | Go modules |

Monorepos are supported — the scanner recursively finds all dependency files.

## Policy File

Create a `policy.yaml` to enforce org-wide rules:

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

Available on [Docker Hub](https://hub.docker.com/r/pawarpoonam/safeupgrade).

## License

MIT
