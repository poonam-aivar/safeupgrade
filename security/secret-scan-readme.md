# Pre-Commit Secret Scanning Setup

## Overview

This folder contains the security infrastructure for automated secret detection in your repository. The setup uses **detect-secrets** by Yelp integrated with **pre-commit hooks** to prevent accidental commits of sensitive information like API keys, passwords, tokens, and other credentials.

---

## 📁 Folder Contents

### Files in This Directory

#### 1. `setup.sh`
**Purpose**: Automated installation and configuration script

**What it does:**
- Installs `pre-commit` framework (if not already installed)
- Installs `detect-secrets` package
- Configures Git hooks for automatic scanning
- Generates or updates the secrets baseline file
- Sets up the repository for team-wide secret detection

**Usage:**
```bash
./security/setup.sh
```

**Requirements:**
- Python 3.x installed
- Git repository initialized
- Internet connection (for package installation)

---

#### 2. `.secrets.baseline`
**Purpose**: Baseline file for known/approved findings

**What it contains:**
- JSON configuration of all enabled secret detection plugins
- Fingerprints of known false positives or approved "secrets"
- Metadata about the scan configuration
- Version information for detect-secrets

**Important Notes:**
- ✅ **MUST be committed to Git** (do NOT add to `.gitignore`)
- Shared across the team to maintain consistency
- Acts as a "whitelist" for approved findings
- Does NOT contain actual secrets, only metadata about scanned files

**When it's updated:**
- When legitimate code triggers false positives
- When new secret types are added/removed
- When scan configuration changes
- When files with approved patterns are added

---

## 🔐 Secret Detection Technology

### What is detect-secrets?

`detect-secrets` is an enterprise-grade secret detection tool created by Yelp that scans code for accidentally committed secrets. It uses multiple detection methods:

#### Detection Plugins (27+ types)

The setup includes detectors for:

**Cloud & Infrastructure:**
- AWS Access Keys
- Azure Storage Keys
- Google Cloud credentials
- IBM Cloud IAM tokens
- Softlayer credentials

**API Keys & Tokens:**
- GitHub tokens
- GitLab tokens
- OpenAI API keys
- Stripe API keys
- SendGrid API keys
- Twilio credentials
- Mailchimp API keys
- NPM tokens
- PyPI tokens
- Slack tokens
- Discord bot tokens
- Telegram bot tokens
- Square OAuth secrets

**Authentication & Secrets:**
- Private SSH/SSL keys
- JWT tokens
- Basic Auth credentials
- Passwords (keyword detection)
- High-entropy strings (Base64, Hex)

**Analysis Methods:**
1. **Keyword Detection**: Looks for patterns like `password=`, `api_key=`, `secret=`
2. **Entropy Analysis**: Detects random-looking strings (likely to be secrets)
3. **Pattern Matching**: Uses regex for known secret formats
4. **Context Analysis**: Considers surrounding code context

---

## 🚀 How It Works

### 1. Initial Setup (One-Time)

When you run `./security/setup.sh`:

```
┌─────────────────────────────────────┐
│  1. Check for pre-commit            │
│  2. Install if missing              │
│  3. Install detect-secrets          │
│  4. Unset existing Git hooks path   │
│  5. Install pre-commit hooks        │
│  6. Configure hooks path            │
│  7. Generate/verify baseline        │
└─────────────────────────────────────┘
```

### 2. Commit Workflow

Every time you commit:

```
git add file.py
git commit -m "message"
       ↓
┌──────────────────────────────┐
│  Pre-commit Hook Triggered   │
└──────────────────────────────┘
       ↓
┌──────────────────────────────┐
│  detect-secrets Scans Files  │
│  - Check against baseline    │
│  - Run all 27+ detectors     │
│  - Analyze entropy           │
│  - Check patterns            │
└──────────────────────────────┘
       ↓
┌──────────────────┬─────────────────┐
│  Secrets Found?  │                 │
└──────────────────┘                 │
    YES ↓              NO ↓           │
┌──────────────┐    ┌──────────────┐ │
│ Block Commit │    │ Allow Commit │ │
│ Show Report  │    │ Success! ✓   │ │
└──────────────┘    └──────────────┘ │
```

---

## 📋 Configuration Details

### Pre-Commit Configuration

**Location**: `/.pre-commit-config.yaml` (repository root)

```yaml
repos:
  - repo: https://github.com/Yelp/detect-secrets
    rev: v1.5.0
    hooks:
      - id: detect-secrets
        args: ['--baseline', 'security/.secrets.baseline']
        stages: [pre-commit]
```

**Key Parameters:**
- `--baseline`: Path to the baseline file for comparison
- `stages: [pre-commit]`: Run on every commit attempt

### Baseline Configuration

**Location**: `/security/.secrets.baseline`

**Key Sections:**

1. **plugins_used**: List of 27+ enabled secret detectors
2. **filters_used**: Heuristic filters to reduce false positives
3. **results**: Known findings (fingerprints of approved "secrets")
4. **generated_at**: Timestamp of last baseline generation

**Filters Applied:**
- ✅ Allowlisted lines (marked with `pragma: allowlist secret`)
- ✅ Indirect references (variable names, not actual secrets)
- ✅ Likely ID strings (UUIDs, hashes)
- ✅ Lock files (package-lock.json, etc.)
- ✅ Non-alphanumeric strings
- ✅ Sequential strings (like "12345")
- ✅ Templated secrets (like `${API_KEY}`)
- ✅ Swagger/OpenAPI files

---

## 🛠️ Usage Guide

### For New Team Members

1. **Clone the repository**
2. **Run setup script** (MANDATORY):
   ```bash
   ./security/setup.sh
   ```
3. **Verify installation**:
   ```bash
   pre-commit run --all-files
   ```

### Daily Development Workflow

**Normal commits work automatically:**
```bash
git add myfile.py
git commit -m "feat: add new feature"
# detect-secrets runs automatically
```

**If secrets are detected:**
```
ERROR: Potential secrets about to be committed to git repo!

Secret Type: Secret Keyword
Location:    config.py:42

Possible mitigations:
  - For information about putting your secrets in a safer place, 
    please ask in #security
  - Mark false positives with an inline `pragma: allowlist secret` comment
```

### Manual Scanning

**Scan all files:**
```bash
pre-commit run detect-secrets --all-files
```

**Scan specific files:**
```bash
pre-commit run detect-secrets --files file1.py file2.js
```

**Generate new baseline:**
```bash
detect-secrets scan > security/.secrets.baseline
```

---

## 🔧 Handling False Positives

### Method 1: Inline Comment (Recommended)

For legitimate code that looks like a secret:

```python
# Example: Documentation or test data
API_ENDPOINT = "https://api.example.com/v1"
EXAMPLE_KEY = "sk-test-1234567890abcdef"  # pragma: allowlist secret
```

### Method 2: Update Baseline

If you have many false positives:

```bash
# Scan and create new baseline with current code
detect-secrets scan > security/.secrets.baseline

# Review the baseline
cat security/.secrets.baseline

# Commit the updated baseline
git add security/.secrets.baseline
git commit -m "chore: update secrets baseline"
```

### Method 3: Audit and Approve

Interactive mode to approve/reject findings:

```bash
detect-secrets audit security/.secrets.baseline
```

**Controls:**
- `y` - Mark as real secret (will be blocked)
- `n` - Mark as false positive (will be allowed)
- `s` - Skip decision
- `q` - Quit

---

## 🐛 Troubleshooting

### Issue: "command not found: pre-commit"

**Solution**: Add Python user bin to PATH

```bash
# For macOS/Linux
export PATH="$HOME/Library/Python/$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2)/bin:$PATH"

# Or add to your shell profile (~/.zshrc or ~/.bashrc)
echo 'export PATH="$HOME/Library/Python/3.13/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Issue: "Cowardly refusing to install hooks with core.hooksPath set"

**Solution**: The setup script handles this automatically, but manual fix:

```bash
git config --unset-all core.hooksPath
pre-commit install
git config core.hooksPath .githooks
```

### Issue: Hooks not running on commit

**Verify installation:**
```bash
# Check pre-commit is installed
pre-commit --version

# Check hooks are installed
ls -la .git/hooks/pre-commit

# Reinstall if needed
./security/setup.sh
```

### Issue: Too many false positives

**Options:**

1. **Adjust entropy limits** in `.secrets.baseline`:
   ```json
   {
     "name": "Base64HighEntropyString",
     "limit": 4.5  // Increase to 5.0 or 6.0
   }
   ```

2. **Exclude specific files** in `.pre-commit-config.yaml`:
   ```yaml
   - id: detect-secrets
     args: ['--baseline', 'security/.secrets.baseline']
     exclude: '^tests/fixtures/.*$'
   ```

3. **Use inline comments** for specific lines

### Issue: Real secret detected, need to remove from history

**If you committed a real secret:**

1. **Immediately rotate the secret** (generate new key/password)
2. **Remove from Git history**:
   ```bash
   # Using git filter-repo (recommended)
   git filter-repo --path file-with-secret --invert-paths
   
   # Or using BFG Repo-Cleaner
   bfg --delete-files file-with-secret
   ```
3. **Force push** (after team coordination):
   ```bash
   git push --force
   ```
4. **Notify security team** immediately

---

## 📊 Best Practices

### ✅ DO:
- Run `./security/setup.sh` immediately after cloning
- Commit the `.secrets.baseline` file to version control
- Use environment variables for secrets (`os.getenv()`)
- Store secrets in secure vaults (AWS Secrets Manager, HashiCorp Vault)
- Use `.env` files for local development (add to `.gitignore`)
- Mark legitimate false positives with `pragma: allowlist secret`
- Keep detect-secrets and pre-commit up to date
- Run manual scans periodically: `pre-commit run --all-files`

### ❌ DON'T:
- Don't add `.secrets.baseline` to `.gitignore`
- Don't disable hooks to "quickly commit" something
- Don't commit actual secrets even temporarily
- Don't use `--no-verify` to bypass hooks (unless emergency)
- Don't hardcode credentials in source code
- Don't assume test/demo keys are safe to commit
- Don't ignore warnings from detect-secrets

---

## 🔄 Maintenance

### Updating detect-secrets

```bash
# Update detect-secrets
pip3 install --upgrade detect-secrets

# Update pre-commit
pip3 install --upgrade pre-commit

# Update pre-commit hooks to latest versions
pre-commit autoupdate

# Test the updates
pre-commit run --all-files
```

### Regenerating Baseline

**When to regenerate:**
- After major codebase refactoring
- When upgrading detect-secrets version
- When changing secret detection plugins

**How to regenerate:**
```bash
# Create fresh baseline
detect-secrets scan > security/.secrets.baseline

# Review the baseline
detect-secrets audit security/.secrets.baseline

# Commit the new baseline
git add security/.secrets.baseline
git commit -m "chore: regenerate secrets baseline"
```

---

## 📚 Additional Resources

### Official Documentation
- **detect-secrets**: https://github.com/Yelp/detect-secrets
- **pre-commit**: https://pre-commit.com/

### Common Commands Reference

```bash
# Run all hooks on all files
pre-commit run --all-files

# Run only detect-secrets hook
pre-commit run detect-secrets --all-files

# Scan for secrets manually
detect-secrets scan

# Audit baseline interactively
detect-secrets audit security/.secrets.baseline

# Update hooks to latest versions
pre-commit autoupdate

# Uninstall hooks (not recommended)
pre-commit uninstall

# Reinstall hooks
pre-commit install
```

---

## 🆘 Support

For any issues, questions, or security concerns:

**Contact**: DevOps Team

**Common Scenarios:**
- Setup fails on your machine
- False positives blocking legitimate commits
- Need to commit sensitive data (discuss alternatives first)
- Accidentally committed a real secret (URGENT)
- Questions about security best practices

---

## 📝 Version Information

- **detect-secrets version**: v1.5.0
- **pre-commit framework**: Latest stable
- **Python requirement**: 3.6+
- **Last updated**: October 2025

---

## 🔒 Security Notes

**This tool is not a silver bullet:**
- It detects common secret patterns but may miss custom formats
- Always follow security best practices
- Use secret management tools for production
- Regular security audits are still necessary
- Educate team members about secure coding practices

**Defense in Depth:**
This pre-commit hook is ONE layer of security. Always use:
- Secret management services (AWS Secrets Manager, Vault)
- Environment variables
- CI/CD secret scanning
- Regular security training
- Code reviews with security focus

---

*Last Updated: October 8, 2025*  
*Maintained by: DevOps Team*

