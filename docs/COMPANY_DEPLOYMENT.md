# Internal Company Deployment Guide

## Overview

This guide explains how to deploy SafeUpgrade Agent within your organization with your own AI models and infrastructure, separate from the open-source public offering.

## Key Separation: Public vs Internal

### Public Open Source Version
- Users bring their own API keys (OpenAI, Anthropic, etc.)
- Public Docker image: `aivar/safeupgrade:latest`
- Free web UI with rate limits
- Community support

### Your Company's Internal Version
- Your organization's AI Gateway (pre-configured API keys)
- Private Docker registry: `your-registry.company.com/safeupgrade:latest`
- Unlimited usage for internal teams
- Internal support via your DevOps team

## Architecture Decision: Two Deployment Modes

### Mode 1: CI/CD Integration (Recommended for most teams)

Teams use SafeUpgrade in their GitHub Actions workflows, pointing to your internal AI Gateway.

**Benefits:**
- No infrastructure for you to maintain
- Teams control when upgrades run
- Fits existing CI/CD workflows
- Minimal setup required

**Setup:**

1. **Create internal configuration template**

```yaml
# .github/workflows/safeupgrade-internal.yml (Template for teams)
name: SafeUpgrade (Internal)

on:
  schedule:
    - cron: '0 6 * * 1'  # Weekly
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write

jobs:
  safeupgrade:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Run SafeUpgrade
        uses: docker://your-registry.company.com/safeupgrade:latest
        with:
          args: upgrade --auto-pr --policy https://internal-config.company.com/safeupgrade-policy.yaml
        env:
          # Your organization's AI Gateway (no personal API key needed!)
          AI_GATEWAY_URL: https://ai-gateway.company.com
          AI_GATEWAY_KEY: ${{ secrets.ORG_AI_GATEWAY_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

2. **Distribute to teams**

Create an internal repository: `company/safeupgrade-workflows`

```
company/safeupgrade-workflows/
├── README.md                          # Usage instructions
├── workflows/
│   ├── safeupgrade-basic.yml         # Weekly scans
│   ├── safeupgrade-aggressive.yml    # Daily scans
│   └── safeupgrade-security-only.yml # CVE fixes only
├── configs/
│   ├── org-policy.yaml               # Organization-wide policy
│   ├── frontend-policy.yaml          # Frontend team overrides
│   └── backend-policy.yaml           # Backend team overrides
└── docs/
    ├── getting-started.md
    └── troubleshooting.md
```

Teams copy the workflow file they need and customize it.

3. **Set up organization secrets**

In GitHub Organization settings:
```bash
# Organization-level secret (all repos can access)
ORG_AI_GATEWAY_KEY = "your-internal-ai-gateway-key"
```

### Mode 2: Centralized Service (For organization-wide control)

You host SafeUpgrade as a service that scans all repos automatically.

**Benefits:**
- Centralized control and monitoring
- Automatic scanning of all repos
- Consistent policy enforcement
- Organization-wide dashboard

**Setup:**

1. **Deploy to Kubernetes**

```yaml
# kubernetes/safeupgrade-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: safeupgrade-controller
  namespace: devops
spec:
  replicas: 3
  selector:
    matchLabels:
      app: safeupgrade
  template:
    metadata:
      labels:
        app: safeupgrade
    spec:
      serviceAccountName: safeupgrade
      containers:
      - name: safeupgrade
        image: your-registry.company.com/safeupgrade:latest
        command: ["safeupgrade", "scan-org", "--org", "your-github-org"]
        env:
        - name: AI_GATEWAY_URL
          value: "https://ai-gateway.company.com"
        - name: AI_GATEWAY_KEY
          valueFrom:
            secretKeyRef:
              name: safeupgrade-secrets
              key: ai-gateway-key
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: safeupgrade-secrets
              key: github-token
        - name: POLICY_FILE
          value: "/config/org-policy.yaml"
        volumeMounts:
        - name: config
          mountPath: /config
      volumes:
      - name: config
        configMap:
          name: safeupgrade-config
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: safeupgrade-weekly-scan
  namespace: devops
spec:
  schedule: "0 6 * * 1"  # Every Monday at 6 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: safeupgrade
          containers:
          - name: scanner
            image: your-registry.company.com/safeupgrade:latest
            command:
            - /bin/sh
            - -c
            - |
              safeupgrade scan-org \
                --org your-github-org \
                --policy /config/org-policy.yaml \
                --report-to-slack
            env:
            - name: AI_GATEWAY_URL
              value: "https://ai-gateway.company.com"
            - name: AI_GATEWAY_KEY
              valueFrom:
                secretKeyRef:
                  name: safeupgrade-secrets
                  key: ai-gateway-key
            - name: SLACK_WEBHOOK
              valueFrom:
                secretKeyRef:
                  name: safeupgrade-secrets
                  key: slack-webhook
          restartPolicy: OnFailure
```

## Building Your Internal Docker Image

### Option A: Pre-configured with your AI Gateway

```dockerfile
# Dockerfile.internal
FROM aivar/safeupgrade:latest

# Add your organization's default configuration
COPY internal-config/.safeupgrade.yaml /etc/safeupgrade/config.yaml

# Pre-configure AI Gateway URL (key still comes from env)
ENV AI_GATEWAY_URL=https://ai-gateway.company.com

# Add your organization's CA certificates if needed
COPY certs/company-ca.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates

# Set default policy
COPY configs/org-policy.yaml /etc/safeupgrade/policy.yaml
```

Build and push:
```bash
docker build -f Dockerfile.internal -t your-registry.company.com/safeupgrade:latest .
docker push your-registry.company.com/safeupgrade:latest
```

### Option B: Fork and customize

If you need deeper customization:

```bash
# Fork the repo
git clone https://github.com/aivar-tech/safeupgrade-agent
cd safeupgrade-agent
git remote add company https://github.company.com/devops/safeupgrade

# Make your changes
# - Add company-specific integrations
# - Customize UI branding
# - Add internal metrics

# Build with your branding
docker build \
  --build-arg VERSION=internal-1.0 \
  -t your-registry.company.com/safeupgrade:internal-1.0 .

docker push your-registry.company.com/safeupgrade:internal-1.0
```

## AI Gateway Configuration

Your AI Gateway should expose an OpenAI-compatible API:

```bash
# Example: Your AI Gateway
curl https://ai-gateway.company.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_ORG_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4.5",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

SafeUpgrade will automatically use your gateway when configured:
```yaml
# .safeupgrade.yaml
ai_backend:
  provider: "gateway"
  endpoint: "https://ai-gateway.company.com"
  api_key: "${AI_GATEWAY_KEY}"
```

## Organization-Wide Policy Management

Host your policy files in a central location:

```yaml
# Hosted at: https://internal-config.company.com/safeupgrade-policy.yaml
global:
  block_canary: true
  block_alpha: true
  enforce_lockfile: true
  max_major_jump: 1
  provenance_required: true

# Critical packages - pinned across all teams
packages:
  react:
    pin_major: 18
    reason: "Company-wide React 19 migration in Q3 2025"
  
  express:
    pin_major: 4
    reason: "Express 5 has breaking auth changes"
  
  # Known compromised packages
  axios:
    block_versions: ["1.14.1"]
    reason: "CVE-2026-12345 - supply chain attack"
  
  "@tanstack/router":
    block_versions: ["1.161.9", "1.161.12"]
    reason: "Malicious code injection - March 2026"

# Alert configuration
alerts:
  compromised_package:
    channel: "#security-incidents"
    teams_webhook: "${TEAMS_WEBHOOK}"
    pagerduty_key: "${PAGERDUTY_KEY}"
    bypass_policy: false  # Block even if policy would allow
    action: "create-incident"
  
  maintainer_change:
    channel: "#security-alerts"
    action: "require-security-review"
    approvers: ["security-team", "engineering-leads"]

# Cost tracking
cost_control:
  monthly_budget: 5000  # USD
  alert_threshold: 0.8
  per_team_limit: 500
```

## Team-Specific Overrides

Teams can extend the org policy:

```yaml
# frontend-team/.safeupgrade.yaml
extends: "https://internal-config.company.com/safeupgrade-policy.yaml"

# Team-specific overrides
packages:
  next:
    allow_minor: true
    block_canary: false  # Frontend team wants canary access
  
  "@emotion/react":
    pin_major: 11
    reason: "UI library migration planned for Sprint 24"
```

## Monitoring & Observability

### Metrics to Track

1. **Upgrade Success Rate**
   - Number of successful vs failed upgrades
   - Common failure reasons

2. **Security Metrics**
   - CVEs detected and fixed
   - Average time to patch vulnerabilities
   - Policy violations by team

3. **Cost Tracking**
   - AI API usage per team
   - Cost per repository
   - Monthly spend trends

### Grafana Dashboard

```yaml
# prometheus-config.yaml
scrape_configs:
  - job_name: 'safeupgrade'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - devops
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: safeupgrade
        action: keep
```

Key metrics to visualize:
- `safeupgrade_scans_total{status="success"}`
- `safeupgrade_upgrades_total{ecosystem="npm"}`
- `safeupgrade_policy_violations_total{package="react"}`
- `safeupgrade_cve_detections_total{severity="critical"}`
- `safeupgrade_ai_api_calls_total{model="claude"}`
- `safeupgrade_ai_cost_usd{team="frontend"}`

## Security Considerations

### API Key Management

**DON'T:**
```yaml
# ❌ Hardcoded in workflow
env:
  AI_GATEWAY_KEY: "sk-1234567890abcdef"  # NEVER DO THIS
```

**DO:**
```yaml
# ✅ Use GitHub Secrets
env:
  AI_GATEWAY_KEY: ${{ secrets.ORG_AI_GATEWAY_KEY }}
```

**BETTER:**
```yaml
# ✅ Use HashiCorp Vault or AWS Secrets Manager
- name: Get secrets from Vault
  uses: hashicorp/vault-action@v2
  with:
    url: https://vault.company.com
    secrets: |
      secret/data/safeupgrade ai_gateway_key | AI_GATEWAY_KEY
```

### Network Isolation

Only allow SafeUpgrade to communicate with:
- Your AI Gateway
- GitHub Enterprise
- Internal policy server
- Package registries (npm, PyPI, Go proxy)

```yaml
# NetworkPolicy example
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: safeupgrade-egress
spec:
  podSelector:
    matchLabels:
      app: safeupgrade
  policyTypes:
  - Egress
  egress:
  # Allow AI Gateway
  - to:
    - podSelector:
        matchLabels:
          app: ai-gateway
    ports:
    - protocol: TCP
      port: 443
  # Allow GitHub Enterprise
  - to:
    - namespaceSelector:
        matchLabels:
          name: github-enterprise
    ports:
    - protocol: TCP
      port: 443
  # Allow internal DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
```

## Rollout Strategy

### Phase 1: Pilot (Week 1-2)
- Select 3-5 pilot teams
- Deploy CI/CD workflow to pilot repos
- Gather feedback
- Tune policies based on false positives

### Phase 2: Early Adopters (Week 3-4)
- Expand to 20% of teams
- Monitor AI costs
- Create runbooks for common issues
- Train team leads

### Phase 3: Company-wide (Week 5-6)
- Deploy to all teams
- Automated onboarding for new repos
- Monthly reporting to leadership

### Phase 4: Optimization (Ongoing)
- Fine-tune policies based on data
- Add integrations (Jira, PagerDuty)
- Optimize AI costs
- Expand to more ecosystems

## Cost Estimation

### AI API Costs

Assumptions:
- 100 repositories
- Average 20 outdated dependencies per repo
- Scanned weekly
- Claude Sonnet 4 ($3 per 1M input tokens, $15 per 1M output tokens)

Calculation:
```
Per scan per repo:
- Input: ~10K tokens (changelogs, CVEs, metadata) * 20 deps = 200K tokens
- Output: ~2K tokens (analysis) * 20 deps = 40K tokens

Per scan cost: (200K * $3 / 1M) + (40K * $15 / 1M) = $0.60 + $0.60 = $1.20

Weekly cost: $1.20 * 100 repos = $120
Monthly cost: $120 * 4 = $480
Annual cost: $480 * 12 = $5,760
```

**Cost savings from prevented vulnerabilities:** $$$$ (1 breach costs more!)

## Support & Troubleshooting

### Common Issues

**Issue: AI Gateway timeout**
```bash
# Check connectivity
curl -H "Authorization: Bearer $AI_GATEWAY_KEY" \
  https://ai-gateway.company.com/health

# Check logs
kubectl logs -n devops deployment/safeupgrade-controller
```

**Issue: Policy not being enforced**
```bash
# Validate policy
safeupgrade policy-check --policy https://internal-config.company.com/safeupgrade-policy.yaml --dry-run

# Check policy fetch
curl https://internal-config.company.com/safeupgrade-policy.yaml
```

**Issue: PRs not being created**
```bash
# Check GitHub token permissions
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://github.company.com/api/v3/user

# Required scopes: repo, workflow
```

### Getting Help

- Internal Slack: `#safeupgrade-support`
- Email: `devops-team@company.com`
- Oncall: PagerDuty integration for P0 issues

## Maintenance

### Monthly Tasks
- Review AI cost trends
- Update organization policy
- Review and merge upstream updates
- Analyze false positive rate

### Quarterly Tasks
- Security audit of deployment
- Review team adoption metrics
- Plan new feature rollouts
- Update documentation

---

## Summary

**For most companies, we recommend Mode 1 (CI/CD Integration):**
- ✅ Easy to deploy
- ✅ Low maintenance
- ✅ Teams control their own upgrades
- ✅ Scales naturally with organization

**Use Mode 2 (Centralized Service) if you need:**
- Organization-wide enforcement
- Automated scanning of all repos
- Centralized dashboard and reporting
- Security team wants full control
