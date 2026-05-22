# Aivar Security Scan Pipeline Template

## Overview

Our **Aivar organization** implements a comprehensive **security scanning CI/CD pipeline** using GitHub Actions. The pipeline automatically scans **`.tf`**, **`.tfvars`**, and **`.py`** files on every Pull Request, ensuring code quality and security compliance before merging.

### Technologies Used
- **Security Scanning**: 
  - **SonarQube** - Code quality and static analysis
  - **Bandit** - Python code security analysis
  - **Trivy** - Container image and Dockerfile vulnerability scanner
  - **Hadolint** - Dockerfile linter and best practices checker
  - **Checkov** - Terraform/Infrastructure security scanning
- **CI/CD**: GitHub Actions with automated security validation
- **Cloud Platform**: AWS (S3 for report storage, OIDC for authentication) 

## Project Structure

**⚠️ MANDATORY**: All files must be organized in the correct folders:

- **Application files** (Python, JavaScript, etc.) → **`app/`** folder
- **Terraform files** (`.tf`, `.tfvars`) → **`Terraform/`** folder
- **Dockerfile** → Root directory (for Trivy and Hadolint scanning)

This organization is required for the security scanning pipeline to function correctly.

```
repo-root/
├── .github/
│   └── workflows/
│       └── security-scan.yml          # Main security scanning CI/CD pipeline
├── .githooks/
│   └── pre-commit                     # Pre-Commit Setup
├── .pre-commit-config.yaml            # Precommit configuration
├── app/                               # Application source code
│   ├── src/                           # Source code files
│   ├── requirements.txt               # Python dependencies
│   └── main.py                        # Main application file
├── Terraform/                         # Infrastructure-as-Code (IaC) files
│   ├── environments/                  # Environment-specific configurations
│   │   ├── dev.tfvars                 # Development environment variables
│   │   └── prod.tfvars                # Production environment variables
│   ├── modules/                       # Reusable Terraform modules
│   │   └── ec2/                       # EC2 instance module
│   ├── main.tf                        # Root Terraform configuration
│   ├── variables.tf                   # Terraform variables
│   └── output.tf                      # Terraform outputs
├── Dockerfile                         # Docker configuration (scanned by Trivy & Hadolint)
├── sonar-project.properties           # SonarQube configuration
├── README.md                          # This documentation
└── .gitignore                         # Git ignore rules
```

### Folder Structure Explanation

#### `.github/workflows/`
- **`security-scan.yml`**: Contains the GitHub Actions workflow that runs security scans
- **Trigger**: Automatically runs on Pull Requests (opened, synchronized, reopened)
- **Purpose**: Ensures code security before merging

#### `app/` Directory
- **Purpose**: Contains all application source code
- **Contents**: Python files, JavaScript files, configuration files, etc.
- **Security Scanning**: 
  - **Bandit**: Scans all `.py` files for Python security vulnerabilities
  - **SonarQube**: Performs static code analysis on all code files
- **Structure**: Organize your application code with subdirectories like `src/`, `config/`, etc.

#### `Terraform/` Directory
- **Purpose**: Contains all Infrastructure-as-Code (IaC) files
- **Contents**: 
  - **`main.tf`**: Root Terraform configuration
  - **`variables.tf`**: Input variables
  - **`output.tf`**: Output values
  - **`environments/`**: Environment-specific variable files (`dev.tfvars`, `prod.tfvars`)
  - **`modules/`**: Reusable Terraform modules (e.g., `ec2/`)
- **Security Scanning**: All `.tf` and `.tfvars` files are scanned by **Checkov** for infrastructure security issues

#### `Dockerfile` (Root Directory)
- **Purpose**: Docker container configuration
- **Security Scanning**: 
  - **Trivy**: Scans for HIGH and CRITICAL severity vulnerabilities in Dockerfile configurations
  - **Hadolint**: Lints Dockerfile for best practices and security misconfigurations

> **⚠️ Important**: This is the **standard Aivar repository folder structure** that all contributors must follow. Deviating from this structure will break the CI/CD pipeline and security scanning.

## Implementation Steps

#### 1. Repository Setup
1. **Create Repository from Aivar Template**:
   - Go to GitHub and create a new repository using **'Aivar Template'**
   - Name your repository and create it

2. **Clone the Repository Locally**:
```bash
# Clone the repository
git clone <repository-url>
cd <repository-name>
```

#### 2. Branch Creation Order (MANDATORY)
**⚠️ CRITICAL**: The following branch creation order must be followed exactly:

1. **Main Branch** (created automatically when repository is created from template)
2. **Develop Branch** (create from main)
3. **Feature Branches** (create from develop)

#### Branch Flow Strategy

#### Why This Logic?
- **Feature Branch Management**: Multiple feature branches can be developed simultaneously
- **Quality Gates**: All features are merged into `develop` for integration testing
- **Production Safety**: Only thoroughly tested code from `develop` reaches `main`
- **Efficiency**: Reduces merge conflicts and ensures systematic deployment

1. **Feature Branches → Develop**: Uses `dev.tfvars` (Development environment)
   - Example: `feature/user-authentication` → `develop`
   - Purpose: Test new features in development environment

2. **Develop → Main**: Uses `prod.tfvars` (Production environment)
   - Example: `develop` → `main`
   - Purpose: Deploy tested features to production

```bash
# Step 1: Ensure you're on main branch
git checkout main

# Step 2: Create develop branch from main
git checkout -b develop
git push origin develop

# Step 3: Create feature branch from develop
git checkout develop
git checkout -b feature/your-feature-name
```

#### 3. Follow Folder Structure
- Place application code in the `app/` directory
- Place Terraform code in the `Terraform/` directory
- Use environment-specific `.tfvars` files
- Follow module structure for reusable components

#### 4. Development Workflow
```bash
# Make your changes
# Follow the folder structure guidelines

# Commit and push
git add .
git commit -m "feat: add new feature"
git push origin feature/your-feature-name
```

#### 5. Create Pull Request
- **⚠️ Prerequisite**: Ensure your repository is added to the AWS OIDC trust relationship (see [AWS OIDC Configuration](#aws-oidc-configuration) section)
- Create PR from `feature/*` to `develop`
- Wait for security scan completion
- Review findings and remediate issues
- Address any PR comments

## AWS OIDC Configuration

### Trust Relationship Setup

**⚠️ IMPORTANT**: When creating a new repository, the pipeline will fail to connect with AWS because the repository needs to be added to the OIDC trust relationship.

The security scanning pipeline uses AWS OIDC (OpenID Connect) to authenticate with AWS. Each repository must be explicitly added to the trust relationship policy.

### How to Add Your Repository

To connect your new repository with AWS, you need to add your repository details to the trust relationship policy in the AWS IAM role `Githubactions`.

#### Step 1: Access AWS IAM Console
1. Go to AWS IAM Console
2. Navigate to **Roles** → **Githubactions**
3. Click on **Trust relationships** tab
4. Click **Edit trust policy**

#### Step 2: Add Your Repository
Add your repository entries to the `"token.actions.githubusercontent.com:sub"` array in the JSON policy:

```json
"token.actions.githubusercontent.com:sub": [
    "repo:YOUR-ORG/YOUR-REPO:ref:refs/heads/main",
    "repo:YOUR-ORG/YOUR-REPO:ref:refs/heads/develop", 
    "repo:YOUR-ORG/YOUR-REPO:ref:refs/heads/feature*",
    "repo:YOUR-ORG/YOUR-REPO:pull_request"
]
```

#### Step 3: Repository Entry Format
Replace `YOUR-ORG/YOUR-REPO` with your actual repository details:

- **`repo:YOUR-ORG/YOUR-REPO:ref:refs/heads/main`** - For main branch
- **`repo:YOUR-ORG/YOUR-REPO:ref:refs/heads/develop`** - For develop branch  
- **`repo:YOUR-ORG/YOUR-REPO:ref:refs/heads/feature*`** - For all feature branches
- **`repo:YOUR-ORG/YOUR-REPO:pull_request`** - For pull requests

#### Step 4: Save Changes
1. Click **Update policy** to save the changes
2. Your repository will now be able to authenticate with AWS

### Example
If your repository is `aivar-tech/my-new-app`, add these entries:
```json
"repo:aivar-tech/my-new-app:ref:refs/heads/main",
"repo:aivar-tech/my-new-app:ref:refs/heads/develop",
"repo:aivar-tech/my-new-app:ref:refs/heads/feature*",
"repo:aivar-tech/my-new-app:pull_request"
```

#### SonarQube Project Configuration

The workflow uses `sonar-project.properties` file in the repository root to configure the SonarQube project key. Ensure this file exists with the correct project key:

```properties
sonar.projectKey="your-project-key"
sonar.sources=.
```

## Security Scan CI/CD Pipeline

### Workflow Overview
The `security-scan.yml` workflow is a comprehensive security scanning pipeline that automatically validates code quality and security on every Pull Request.

### Workflow Trigger
The security scanning pipeline is automatically triggered on **Pull Requests**:
- **Opened**: When a new PR is created
- **Synchronized**: When new commits are pushed to the PR
- **Reopened**: When a closed PR is reopened

### Environment Detection Logic

The pipeline intelligently detects the target environment based on branch patterns:

```yaml
if: |
  (startsWith(github.head_ref, 'feature/') && github.base_ref == 'develop') ||
  (github.head_ref == 'develop' && github.base_ref == 'main')
```

### Pipeline Execution Flow

1. **Code Checkout**: Repository code is checked out
2. **AWS Authentication**: Configured using OIDC (OpenID Connect) with IAM role-based authentication
3. **Tool Installation**: 
   - SonarScanner CLI (for SonarQube)
   - Bandit (Python security scanner)
   - Trivy (Dockerfile vulnerability scanner)
   - Hadolint (Dockerfile linter)
   - Checkov (Terraform security scanner)
4. **Environment Detection**: Determines which `.tfvars` file to use based on branch pattern
5. **Security Scans** (run in parallel where possible):
   - **SonarQube**: Static code analysis on all code files
   - **Bandit**: Scans all Python files in the repository
   - **Trivy**: Scans Dockerfile for HIGH and CRITICAL vulnerabilities
   - **Hadolint**: Lints Dockerfile for best practices
   - **Checkov**: Scans Terraform plan (if Terraform folder exists)
     - `terraform init`: Initialize Terraform
     - `terraform plan`: Generate execution plan with appropriate `.tfvars` file
     - Convert plan to JSON for Checkov analysis
6. **Report Generation**: Creates comprehensive security report in Markdown format
7. **S3 Upload**: Stores reports in AWS S3 bucket (`aivar-githubactions-reports`) for audit trail
8. **PR Comments**: 
   - Posts summary comment to PR with total issue count and S3 report link
   - Posts inline review comments for issues found in PR diff (if total issues ≤ 100)
   - If total issues > 100, inline comments are skipped to avoid rate limits (full report available in S3)

## Working with Pull Requests

### After Creating a Pull Request

#### 1. Automated Workflow Execution
- The security scan workflow runs automatically
- Progress is visible in the **GitHub Actions** tab
- Real-time status updates are provided

#### 2. Security Report Access
- **S3 Storage**: All reports are stored in `s3://aivar-githubactions-reports/security-reports/{username}/{repo-name}/{branch-name}/`
- **Direct Links**: Reports are accessible via AWS Console and direct download links
- **Report Types**:
  - `{commit-id}-report.md`: Human-readable summary report (uploaded to S3)
  - `sonar_issues.json`: SonarQube code quality findings
  - `bandit-report.json`: Python security findings
  - `trivy-report.json`: Dockerfile vulnerability findings
  - `hadolint-report.json`: Dockerfile linting findings
  - `checkov-report.json`: Infrastructure security findings (if Terraform exists)

#### 3. PR Comment Integration
- **Summary Comment**: A comprehensive summary is posted to the PR with:
  - Total issue count breakdown by scanner
  - Direct link to full report in S3
  - Truncation notice if issues exceed 100 (to avoid rate limits)
- **Inline Review Comments**: 
  - Posted for issues found in PR diff (only for lines changed in the PR)
  - **Note**: Inline comments are automatically skipped if total issues > 100 to avoid GitHub API rate limits
  - Each comment includes a link to the exact line of code
  - Contextual information: Severity, confidence, scanner type, and remediation guidance
- **Rate Limit Protection**: The workflow implements intelligent batching and retry logic to handle GitHub API rate limits gracefully

## Troubleshooting

### Common Issues

#### 1. Pipeline Failures
- **Check folder structure**: Ensure files are in correct locations (`app/` for code, `Terraform/` for IaC, `Dockerfile` in root)
- **Verify AWS OIDC permissions**: Ensure repository is added to IAM role trust relationship
- **Review logs**: Check GitHub Actions logs for specific errors
- **Summary report not generated**: If there are too many errors, the summary report may not be generated in the PR. In this case, view the detailed reports in the S3 bucket - the location is provided in the GitHub Actions tab

#### 2. Inline Comments Not Appearing
- **High issue count**: If total issues > 100, inline comments are automatically skipped to avoid rate limits. View the full report in S3.
- **Issues not in PR diff**: Only issues in files/lines changed in the PR are posted as inline comments
- **Rate limit errors**: The workflow includes retry logic, but if issues persist, check GitHub API rate limit status

### Getting Help

If any help is needed, contact the **DevOps team**.

---

*This repository follows the Aivar standard for secure, compliant, and maintainable infrastructure code.*
