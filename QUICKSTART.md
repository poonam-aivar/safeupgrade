# Quick Start Guide

## For Open Source Users (Anyone)

### 1. Using Docker (Easiest)

```bash
# Scan your project
docker run -v $(pwd):/repo \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  aivar/safeupgrade:latest \
  scan --repo /repo

# Upgrade dependencies
docker run -v $(pwd):/repo \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  aivar/safeupgrade:latest \
  upgrade --repo /repo --auto-pr
```

### 2. Using GitHub Actions

Add to `.github/workflows/safeupgrade.yml`:

```yaml
name: SafeUpgrade

on:
  schedule:
    - cron: '0 6 * * 1'  # Weekly on Monday
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write

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

Add secrets in GitHub repo settings:
- `OPENAI_API_KEY` - Your OpenAI API key
- `GITHUB_TOKEN` - Already available automatically

### 3. Install CLI

```bash
# macOS/Linux
brew install aivar/tap/safeupgrade

# Or download binary
curl -sSL https://get.safeupgrade.io | sh

# Run it
cd your-project
safeupgrade scan
```

## For Your Company (Internal)

### 1. Build Internal Image

```bash
# Clone and customize
git clone https://github.com/aivar-tech/safeupgrade-agent
cd safeupgrade-agent

# Create Dockerfile.internal
cat > Dockerfile.internal <<EOF
FROM aivar/safeupgrade:latest

# Your AI Gateway (pre-configured)
ENV AI_GATEWAY_URL=https://ai-gateway.yourcompany.com

# Your org policy
COPY configs/your-org-policy.yaml /etc/safeupgrade/policy.yaml
EOF

# Build and push
docker build -f Dockerfile.internal \
  -t your-registry.company.com/safeupgrade:latest .

docker push your-registry.company.com/safeupgrade:latest
```

### 2. Create Workflow Template

Create `company/safeupgrade-workflows/workflow-template.yml`:

```yaml
name: SafeUpgrade (Internal)

on:
  schedule:
    - cron: '0 6 * * 1'
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write

jobs:
  upgrade:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: docker://your-registry.company.com/safeupgrade:latest
        with:
          args: upgrade --auto-pr
        env:
          # Your organization's AI Gateway
          AI_GATEWAY_KEY: ${{ secrets.ORG_AI_GATEWAY_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 3. Set Organization Secret

In GitHub Organization Settings → Secrets:
- Add `ORG_AI_GATEWAY_KEY` = your internal AI gateway API key

### 4. Rollout to Teams

Teams copy the workflow template to their repos:

```bash
# Teams do this:
mkdir -p .github/workflows
cp company/safeupgrade-workflows/workflow-template.yml \
   .github/workflows/safeupgrade.yml

# Commit and push
git add .github/workflows/safeupgrade.yml
git commit -m "chore: add SafeUpgrade workflow"
git push
```

That's it! The workflow will run weekly and create PRs automatically.

## Configuration Options

### Basic Config (.safeupgrade.yaml)

```yaml
ai_backend:
  provider: "openai"  # or anthropic, bedrock, ollama, gateway
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o"

github:
  token: "${GITHUB_TOKEN}"
  auto_pr: true

policy:
  file: "configs/policy.yaml"

upgrade:
  strategy: "safe"  # only upgrade safe dependencies
  run_tests: true
  rollback_on_failure: true
```

### Policy File (configs/policy.yaml)

```yaml
global:
  block_canary: true      # Block pre-release versions
  block_alpha: true
  max_major_jump: 1       # Allow max 1 major version jump

packages:
  # Pin specific packages
  react:
    pin_major: 18
    reason: "React 19 migration planned"
  
  # Block compromised versions
  axios:
    block_versions: ["1.14.1"]
    reason: "Supply chain attack"
```

## Common Use Cases

### 1. Weekly Security Patches Only

```yaml
# .safeupgrade.yaml
upgrade:
  strategy: "safe"

security:
  check_cves: true
  block_vulnerable: true
```

### 2. Aggressive Updates (Patch + Minor)

```yaml
global:
  max_major_jump: 0  # Block major versions
  
upgrade:
  strategy: "all"  # Upgrade all safe dependencies
```

### 3. Manual Review for Major Versions

```yaml
global:
  max_major_jump: 0

packages:
  react:
    pin_major: 18
    reason: "Manual review required for React 19"
```

## Troubleshooting

### "AI API key not set"

```bash
# For OpenAI
export OPENAI_API_KEY=sk-...

# For Anthropic
export ANTHROPIC_API_KEY=sk-ant-...

# For your company gateway
export AI_GATEWAY_KEY=your-key
export AI_GATEWAY_URL=https://ai-gateway.yourcompany.com
```

### "Tests failed after upgrade"

SafeUpgrade automatically rolls back failed upgrades:

```yaml
upgrade:
  run_tests: true
  rollback_on_failure: true  # Automatic rollback
```

### "Too many policy violations"

Review your policy file - it might be too strict:

```bash
# Check what's being blocked
safeupgrade scan --policy configs/policy.yaml --verbose

# Test policy changes
safeupgrade policy-check --policy configs/policy.yaml --dry-run
```

## Next Steps

1. **Try it on a test project first**
2. **Review the generated PR** - SafeUpgrade explains each upgrade
3. **Tune your policy** based on false positives
4. **Roll out to more repos**

## Getting Help

- 📚 [Full Documentation](https://docs.safeupgrade.io)
- 💬 [GitHub Discussions](https://github.com/aivar-tech/safeupgrade-agent/discussions)
- 🐛 [Report Issues](https://github.com/aivar-tech/safeupgrade-agent/issues)
- 🔐 [Security Policy](SECURITY.md)

---

**Ready to make your dependencies safer?** 🚀
