# Enterprise Deployment Guide

## Internal Company Deployment

This guide explains how to deploy SafeUpgrade Agent within your organization with custom AI models and internal infrastructure.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Your Organization                     │
│                                                          │
│  ┌──────────────┐      ┌─────────────────┐             │
│  │  GitHub      │      │  Internal AI    │             │
│  │  Enterprise  │◄────►│  Gateway        │             │
│  │              │      │  (Your Models)  │             │
│  └──────────────┘      └─────────────────┘             │
│         ▲                       ▲                       │
│         │                       │                       │
│         │              ┌────────┴────────┐              │
│         │              │                 │              │
│         │              │  SafeUpgrade    │              │
│         └──────────────┤  Agent          │              │
│                        │  (Self-Hosted)  │              │
│                        └─────────────────┘              │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Deployment Options

### Option 1: Kubernetes (Recommended)

#### Prerequisites
- Kubernetes cluster (1.24+)
- kubectl configured
- Helm 3.x
- Internal AI Gateway URL and API key

#### Helm Chart Installation

Create `values.yaml`:

```yaml
# values.yaml
image:
  repository: your-registry.company.com/safeupgrade
  tag: "latest"
  pullPolicy: Always

config:
  aiBackend:
    provider: "gateway"
    gatewayURL: "https://ai-gateway.internal.company.com"
    gatewayKey: "your-internal-api-key"  # Use secret instead
  
  github:
    enterpriseURL: "https://github.company.com"
    apiURL: "https://github.company.com/api/v3"
  
  policy:
    defaultFile: "/etc/safeupgrade/policy.yaml"

secrets:
  aiGatewayKey: "changeme"  # Override via --set or external secret
  githubToken: "changeme"

resources:
  limits:
    cpu: "2"
    memory: "4Gi"
  requests:
    cpu: "500m"
    memory: "1Gi"

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilization: 70

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: safeupgrade.company.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: safeupgrade-tls
      hosts:
        - safeupgrade.company.com

persistence:
  enabled: true
  storageClass: "fast-ssd"
  size: 50Gi

monitoring:
  prometheus:
    enabled: true
  grafana:
    enabled: true
```

Install:
```bash
helm install safeupgrade ./helm/safeupgrade \
  --namespace safeupgrade \
  --create-namespace \
  --values values.yaml \
  --set secrets.aiGatewayKey=$AI_GATEWAY_KEY \
  --set secrets.githubToken=$GITHUB_TOKEN
```

### Option 2: Docker Compose (Development/Small Teams)

```yaml
# docker-compose.yml
version: '3.8'

services:
  safeupgrade-api:
    image: aivar/safeupgrade:latest
    command: ["serve", "--port", "8080"]
    ports:
      - "8080:8080"
    environment:
      - AI_GATEWAY_URL=https://ai-gateway.internal.company.com
      - AI_GATEWAY_KEY=${AI_GATEWAY_KEY}
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - GITHUB_ENTERPRISE_URL=https://github.company.com
      - POLICY_FILE=/config/policy.yaml
    volumes:
      - ./configs:/config:ro
      - ./data:/data
    restart: unless-stopped
    networks:
      - safeupgrade

  safeupgrade-worker:
    image: aivar/safeupgrade:latest
    command: ["worker"]
    environment:
      - AI_GATEWAY_URL=https://ai-gateway.internal.company.com
      - AI_GATEWAY_KEY=${AI_GATEWAY_KEY}
      - REDIS_URL=redis://redis:6379
    volumes:
      - ./configs:/config:ro
    depends_on:
      - redis
    restart: unless-stopped
    networks:
      - safeupgrade

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    networks:
      - safeupgrade

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=safeupgrade
      - POSTGRES_USER=safeupgrade
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - safeupgrade

volumes:
  redis-data:
  postgres-data:

networks:
  safeupgrade:
```

### Option 3: GitHub Actions Self-Hosted Runners

```yaml
# .github/workflows/safeupgrade-internal.yml
name: SafeUpgrade (Internal)

on:
  schedule:
    - cron: '0 6 * * 1'
  workflow_dispatch:

jobs:
  safeupgrade:
    runs-on: [self-hosted, linux, x64]
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Run SafeUpgrade
        uses: docker://your-registry.company.com/safeupgrade:latest
        with:
          args: upgrade --policy configs/policy.yaml
        env:
          AI_GATEWAY_URL: https://ai-gateway.internal.company.com
          AI_GATEWAY_KEY: ${{ secrets.AI_GATEWAY_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Create PR
        if: success()
        run: |
          gh pr create \
            --title "chore(deps): SafeUpgrade automated update" \
            --body-file upgrade_report.json \
            --base main
```

## Configuration Management

### Centralized Policy for Organization

```yaml
# configs/org-policy.yaml
global:
  block_canary: true
  block_alpha: true
  enforce_lockfile: true
  max_major_jump: 1  # Allow one major version jump
  provenance_required: true
  
  # Organization-wide blocked versions
  blocked_versions:
    - "*-evil*"
    - "*-compromised*"

# Critical infrastructure packages
packages:
  # Pin production dependencies
  react:
    pin_major: 18
    reason: "React 19 migration planned for Q3 2025"
  
  express:
    pin_major: 4
    reason: "Express 5 has breaking changes"
  
  # Known compromised packages
  axios:
    block_versions: ["1.14.1"]
    reason: "Supply chain attack - CVE-2026-12345"
  
  "@tanstack/router":
    block_versions: ["1.161.9", "1.161.12"]
    reason: "Malicious code injection - March 2026"

# Team-specific overrides
teams:
  frontend:
    packages:
      next:
        allow_minor: true
        block_canary: true
  
  backend:
    packages:
      fastapi:
        allow_minor: true
        pin_major: 0  # 0.x versions allowed

alerts:
  compromised_package:
    channel: "#security-alerts"
    teams_webhook: "${TEAMS_WEBHOOK_URL}"
    pagerduty_key: "${PAGERDUTY_KEY}"
    bypass_policy: true
    action: "auto-pin-previous"
  
  maintainer_change:
    channel: "#security-alerts"
    action: "block-until-reviewed"
    require_approvers: ["security-team", "engineering-leads"]
```

### Environment-Specific Configuration

```bash
# .env.production
AI_GATEWAY_URL=https://ai-gateway.internal.company.com
AI_GATEWAY_KEY=your-internal-key
GITHUB_ENTERPRISE_URL=https://github.company.com
POLICY_FILE=/etc/safeupgrade/org-policy.yaml
LOG_LEVEL=info
METRICS_ENABLED=true
PROMETHEUS_PORT=9090
```

## Security Best Practices

### 1. API Key Management

Use Kubernetes secrets or HashiCorp Vault:

```bash
# Create secret
kubectl create secret generic safeupgrade-secrets \
  --from-literal=ai-gateway-key="your-key" \
  --from-literal=github-token="your-token" \
  --namespace safeupgrade

# Reference in deployment
env:
  - name: AI_GATEWAY_KEY
    valueFrom:
      secretKeyRef:
        name: safeupgrade-secrets
        key: ai-gateway-key
```

### 2. Network Isolation

```yaml
# NetworkPolicy example
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: safeupgrade-network-policy
spec:
  podSelector:
    matchLabels:
      app: safeupgrade
  policyTypes:
    - Ingress
    - Egress
  egress:
    - to:
      - namespaceSelector:
          matchLabels:
            name: ai-gateway
      ports:
        - protocol: TCP
          port: 443
    - to:
      - namespaceSelector:
          matchLabels:
            name: github-enterprise
      ports:
        - protocol: TCP
          port: 443
```

### 3. RBAC Configuration

```yaml
# ServiceAccount with minimal permissions
apiVersion: v1
kind: ServiceAccount
metadata:
  name: safeupgrade
  namespace: safeupgrade
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: safeupgrade-role
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: safeupgrade-rolebinding
subjects:
  - kind: ServiceAccount
    name: safeupgrade
roleRef:
  kind: Role
  name: safeupgrade-role
  apiGroup: rbac.authorization.k8s.io
```

## Monitoring & Observability

### Prometheus Metrics

```go
// Add to main.go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    upgradesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "safeupgrade_upgrades_total",
            Help: "Total number of upgrades attempted",
        },
        []string{"status", "ecosystem"},
    )
    
    policyViolations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "safeupgrade_policy_violations_total",
            Help: "Total policy violations detected",
        },
        []string{"package", "reason"},
    )
)
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "SafeUpgrade Metrics",
    "panels": [
      {
        "title": "Upgrades by Status",
        "targets": [
          {
            "expr": "rate(safeupgrade_upgrades_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## Multi-Organization Support

### Database Schema for Multi-Tenancy

```sql
-- organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    ai_gateway_url VARCHAR(500),
    ai_gateway_key_encrypted TEXT,
    github_org VARCHAR(255),
    policy_config JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- repositories table
CREATE TABLE repositories (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    github_url VARCHAR(500),
    last_scan TIMESTAMP,
    status VARCHAR(50),
    UNIQUE(org_id, name)
);

-- scan_results table
CREATE TABLE scan_results (
    id UUID PRIMARY KEY,
    repo_id UUID REFERENCES repositories(id),
    scan_date TIMESTAMP DEFAULT NOW(),
    outdated_count INT,
    vulnerable_count INT,
    report_json JSONB
);
```

## Cost Optimization

### AI Model Usage Tracking

```yaml
# Add to config
cost_control:
  monthly_budget: 1000  # USD
  alert_threshold: 0.8  # 80%
  per_repo_limit: 50    # Max API calls per repo per month
  cache_ttl: 3600       # Cache AI results for 1 hour
```

## Troubleshooting

### Common Issues

1. **AI Gateway Connection Failed**
```bash
# Test connectivity
curl -H "Authorization: Bearer $AI_GATEWAY_KEY" \
  https://ai-gateway.internal.company.com/health

# Check DNS resolution
nslookup ai-gateway.internal.company.com
```

2. **Policy Violations Not Enforced**
```bash
# Validate policy file
./safeupgrade policy-check --policy configs/org-policy.yaml --dry-run
```

3. **GitHub Enterprise API Issues**
```bash
# Test GitHub connectivity
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://github.company.com/api/v3/user
```

## Support & Maintenance

### Backup Strategy
- Daily automated backups of policy configs
- Scan history retention: 90 days
- Audit logs: 1 year

### Update Process
1. Test new version in staging environment
2. Review changelog for breaking changes
3. Update Helm chart version
4. Rolling update in production
5. Monitor metrics for 24 hours

## Contact

- Internal Slack: #safeupgrade-support
- Email: safeupgrade-team@company.com
- Incident Response: PagerDuty integration enabled
