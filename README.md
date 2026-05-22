# Pre-Commit Security Template


## 🚨 **IMPORTANT: MANDATORY FIRST STEP** 🚨

### **AFTER CLONING THIS REPOSITORY, YOU MUST RUN:**

```bash
./security/setup.sh
```

**This step is REQUIRED before making any commits!**

---

> **Note:** Feel free to customize this README file to match your project's needs after completing the setup.

---

## About This Template

This repository template provides automated secret detection for your code commits.

## What the Setup Script Does

Running `./security/setup.sh` will:
- Install pre-commit hooks
- Install detect-secrets
- Configure git hooks
- Generate secrets baseline

## What It Protects Against

Automatically scans your commits for:
- AWS Keys
- GitHub Tokens
- API Keys (Stripe, SendGrid, etc.)
- Private Keys
- Passwords
- High-entropy strings
- And 27+ other secret types

**If secrets are detected, the commit will be blocked.**


## Support

For any inquiries or problems, kindly contact the DevOps team.

