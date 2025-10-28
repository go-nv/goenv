# What's New in Documentation

Recent documentation improvements to help you get more out of goenv.

**Last Updated:** October 2025

## 🎉 New Comprehensive Guides

### [Hooks System Quick Start](./HOOKS_QUICKSTART.md) ⭐ NEW

**Get productive with hooks in 5 minutes**

- Setup guide with copy-paste examples
- Common use cases (logging, notifications, compliance)
- Security best practices (SSRF, shell injection prevention)
- SOC 2 and SBOM automation examples
- Complete YAML configuration reference

**Perfect for:** Anyone wanting to automate goenv workflows

**Quick start:**
```bash
goenv hooks init
# Edit ~/.goenv/hooks.yaml
goenv hooks validate
```

---

### [Compliance Use Cases Guide](./COMPLIANCE_USE_CASES.md) ⭐ NEW

**Complete compliance and security reference**

- SOC 2 Type II compliance examples
- ISO 27001 asset management
- Vulnerability management integration
- SBOM generation (CycloneDX, SPDX)
- Change management workflows
- Multi-environment consistency checks

**Perfect for:** Compliance teams, security engineers, enterprise users

**Quick start:**
```bash
# Generate inventory
goenv inventory go --json --checksums > inventory.json

# Generate SBOM
goenv sbom project --tool=cyclonedx-gomod --output=sbom.json
```

---

### [Platform Support Matrix](./PLATFORM_SUPPORT.md) ⭐ NEW

**Know exactly what works on your platform**

- Complete feature compatibility matrix
- Platform-specific behaviors (macOS, Linux, Windows, WSL)
- Architecture support (AMD64, ARM64, ARMv6/7, and more)
- Shell compatibility chart
- CI/CD platform support
- Performance characteristics per platform

**Perfect for:** Multi-platform teams, CI/CD engineers

**Covers:** Linux, macOS, Windows, FreeBSD, Docker, WSL, ARM64, Rosetta 2

---

### [Modern vs Legacy Commands](./MODERN_COMMANDS.md) ⭐ NEW

**Learn the recommended command interface**

- Modern commands (`use`, `current`, `list`) vs legacy (`local`, `global`, `versions`)
- Why modern commands are better
- Migration guide for existing scripts
- Feature comparison matrix
- Team onboarding tips

**Perfect for:** New users, teams standardizing workflows

**Quick comparison:**
```bash
# Modern (recommended)
goenv use 1.25.2              # Set version
goenv current                 # Check active version
goenv list --remote           # Browse versions

# Legacy (still works, but not recommended for new code)
goenv local 1.25.2
goenv version
goenv install --list
```

---

### [Cache Troubleshooting Guide](./CACHE_TROUBLESHOOTING.md) ⭐ NEW

**Solve cache issues fast**

- All cache types explained
- Common issues and solutions
- Migration from bash to Go implementation
- Architecture-specific cache conflicts
- Performance optimization
- Diagnostic commands

**Perfect for:** Users experiencing cache issues, migrating from bash goenv

**Quick diagnostics:**
```bash
goenv doctor                   # Overall health
goenv cache status             # Cache info
goenv cache clean all --force  # Nuclear option
```

---

### [JSON Output Guide](./JSON_OUTPUT_GUIDE.md) ⭐ NEW

**Automate everything with structured output**

- Complete JSON schema reference
- CI/CD integration examples
- Parsing examples (jq, Python, Node.js)
- Error handling patterns
- GitHub Actions, GitLab CI, Jenkins examples

**Perfect for:** Automation engineers, CI/CD pipelines, tool developers

**Quick example:**
```bash
# Get installed versions as JSON
goenv list --json

# Parse with jq
goenv list --json | jq -r '.[] | select(.active) | .version'
```

## 🔄 Major Documentation Updates

### Enhanced: Command Reference

**Location:** [docs/reference/COMMANDS.md](./reference/COMMANDS.md)

**What's new:**
- ✅ `--force` guidance for non-interactive commands
- ✅ New `goenv vscode setup` unified command
- ✅ Enhanced `goenv cache clean` documentation
- ✅ Enhanced `goenv uninstall` with CI/CD examples

**Why it matters:** Better CI/CD integration, faster VS Code setup

---

### Enhanced: VS Code Integration

**Location:** [docs/user-guide/VSCODE_INTEGRATION.md](./user-guide/VSCODE_INTEGRATION.md)

**What's new:**
- ✅ `goenv vscode setup` as primary option
- ✅ One-command setup (init + sync + doctor)
- ✅ Clear mode selection guidance
- ✅ Troubleshooting improvements

**Why it matters:** Get VS Code working with goenv in one command

**Quick start:**
```bash
cd ~/my-project
goenv vscode setup
```

---

### Enhanced: Hooks Security

**Location:** [docs/HOOKS.md](./HOOKS.md)

**What's new:**
- ✅ Stronger `run_command` security guidance
- ✅ Args array pattern shown first
- ✅ Clearer shell injection warnings
- ✅ Security-first examples

**Why it matters:** Prevent shell injection vulnerabilities

---

### Enhanced: Documentation Structure

**Location:** [docs/README.md](./README.md) and main [README.md](../README.md)

**What's new:**
- ✅ Reorganized by user type
- ✅ All new guides prominently linked
- ✅ Clear categorization (Getting Started, Reference, Advanced)
- ✅ "See Also" cross-references throughout

**Why it matters:** Find what you need faster

## 🎯 New Documentation Features

### Documentation Review Checklist

**Location:** [DOCUMENTATION_REVIEW_CHECKLIST.md](./DOCUMENTATION_REVIEW_CHECKLIST.md)

**What it is:** Comprehensive checklist for documentation PRs

**Use it when:**
- Contributing new documentation
- Reviewing documentation PRs
- Ensuring quality standards

---

### Documentation Contribution Guide

**Location:** [CONTRIBUTING.md](./CONTRIBUTING.md#documentation-contributions)

**What it is:** Complete guide to contributing documentation

**Includes:**
- Documentation standards and templates
- Where to add documentation
- Style guide and patterns
- Testing documentation
- Examples of excellent docs

**Perfect for:** First-time contributors, documentation reviewers

---

### Quick Reference Card

**Location:** [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) ⭐ NEW

**What it is:** One-page cheat sheet for common commands

**Perfect for:** Printing, quick lookup, new users

---

### FAQ

**Location:** [FAQ.md](./FAQ.md) ⭐ NEW

**What it is:** Frequently asked questions with quick answers

**Perfect for:** Self-service troubleshooting, common questions

## 📋 Documentation by User Type

### New Users

**Start here:**
1. [Installation Guide](./user-guide/INSTALL.md)
2. [Modern Commands Guide](./MODERN_COMMANDS.md) - Learn the right way
3. [Quick Reference](./QUICK_REFERENCE.md) - Cheat sheet
4. [FAQ](./FAQ.md) - Common questions

### Existing Users

**Check out:**
1. [What's New](./WHATS_NEW_DOCUMENTATION.md) - This document!
2. [Modern Commands](./MODERN_COMMANDS.md) - Update your workflows
3. [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md) - Solve issues
4. [JSON Output Guide](./JSON_OUTPUT_GUIDE.md) - Automate more

### Compliance Teams

**Essential reading:**
1. [Compliance Use Cases](./COMPLIANCE_USE_CASES.md) - SOC 2, ISO 27001
2. [SBOM Command Reference](./reference/COMMANDS.md#goenv-sbom)
3. [Inventory Command Reference](./reference/COMMANDS.md#goenv-inventory)
4. [Hooks Quick Start](./HOOKS_QUICKSTART.md) - Automation

### CI/CD Engineers

**Must read:**
1. [CI/CD Guide](./CI_CD_GUIDE.md)
2. [JSON Output Guide](./JSON_OUTPUT_GUIDE.md) - Automation
3. [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md) - Optimization
4. [Platform Support](./PLATFORM_SUPPORT.md) - Cross-platform

### Security Engineers

**Security-focused:**
1. [Hooks Security Model](./HOOKS.md#security-model)
2. [Compliance Use Cases](./COMPLIANCE_USE_CASES.md)
3. [Platform Support](./PLATFORM_SUPPORT.md) - Security features
4. [run_command Security](./HOOKS.md#run_command)

### Platform Engineers

**Platform-specific:**
1. [Platform Support Matrix](./PLATFORM_SUPPORT.md)
2. [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md)
3. [Cross-Building Guide](./advanced/CROSS_BUILDING.md)
4. [What NOT to Sync](./advanced/WHAT_NOT_TO_SYNC.md)

## 🔍 Finding Documentation

### By Topic

**Installation & Setup:**
- [Installation Guide](./user-guide/INSTALL.md)
- [How It Works](./user-guide/HOW_IT_WORKS.md)
- [VS Code Integration](./user-guide/VSCODE_INTEGRATION.md)

**Commands:**
- [Command Reference](./reference/COMMANDS.md)
- [Modern Commands Guide](./MODERN_COMMANDS.md)
- [Quick Reference](./QUICK_REFERENCE.md)

**Automation:**
- [Hooks Quick Start](./HOOKS_QUICKSTART.md)
- [Hooks (Complete)](./HOOKS.md)
- [JSON Output Guide](./JSON_OUTPUT_GUIDE.md)
- [CI/CD Guide](./CI_CD_GUIDE.md)

**Compliance:**
- [Compliance Use Cases](./COMPLIANCE_USE_CASES.md)
- [SBOM Command](./reference/COMMANDS.md#goenv-sbom)
- [Inventory Command](./reference/COMMANDS.md#goenv-inventory)

**Troubleshooting:**
- [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md)
- [Platform Support](./PLATFORM_SUPPORT.md)
- [FAQ](./FAQ.md)

**Advanced:**
- [Smart Caching](./advanced/SMART_CACHING.md)
- [Cross-Building](./advanced/CROSS_BUILDING.md)
- [GOPATH Integration](./advanced/GOPATH_INTEGRATION.md)
- [What NOT to Sync](./advanced/WHAT_NOT_TO_SYNC.md)

## 💡 Documentation Highlights

### Most Popular New Content

1. **Hooks Quick Start** - Fastest growing
2. **JSON Output Guide** - Most requested
3. **Platform Support Matrix** - Most comprehensive
4. **Modern Commands** - Best for beginners
5. **Compliance Use Cases** - Enterprise favorite

### Best Examples

**Want to write great docs? Learn from these:**

- [Hooks Quick Start](./HOOKS_QUICKSTART.md) - Perfect quick start pattern
- [goenv vscode setup](./reference/COMMANDS.md#goenv-vscode-setup) - Clear command docs
- [run_command action](./HOOKS.md#run_command) - Security-first approach
- [Compliance Use Cases](./COMPLIANCE_USE_CASES.md) - Comprehensive how-to

### Hidden Gems

**Lesser-known but valuable:**

- [JSON schemas](./JSON_OUTPUT_GUIDE.md#json-schemas) - For tool developers
- [Cache types](./CACHE_TROUBLESHOOTING.md#cache-types) - Understanding internals
- [Shell support matrix](./PLATFORM_SUPPORT.md#shell-support) - Platform compatibility
- [Documentation patterns](./CONTRIBUTING.md#common-documentation-patterns) - For contributors

## 🎓 Learning Paths

### Path 1: Complete Beginner

1. [Installation Guide](./user-guide/INSTALL.md) - Get goenv installed
2. [Quick Reference](./QUICK_REFERENCE.md) - Learn basic commands
3. [Modern Commands](./MODERN_COMMANDS.md) - Use recommended interface
4. [VS Code Integration](./user-guide/VSCODE_INTEGRATION.md) - Set up your editor
5. [FAQ](./FAQ.md) - Common questions answered

**Time:** ~1 hour

---

### Path 2: Automation Engineer

1. [JSON Output Guide](./JSON_OUTPUT_GUIDE.md) - Structured output
2. [Hooks Quick Start](./HOOKS_QUICKSTART.md) - Automate workflows
3. [CI/CD Guide](./CI_CD_GUIDE.md) - Pipeline integration
4. [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md) - Optimize performance

**Time:** ~2 hours

---

### Path 3: Compliance Professional

1. [Compliance Use Cases](./COMPLIANCE_USE_CASES.md) - Overview
2. [SBOM Command](./reference/COMMANDS.md#goenv-sbom) - Generate SBOMs
3. [Inventory Command](./reference/COMMANDS.md#goenv-inventory) - Track installations
4. [Hooks System](./HOOKS_QUICKSTART.md) - Automate audit trails

**Time:** ~1.5 hours

---

### Path 4: Platform Engineer

1. [Platform Support Matrix](./PLATFORM_SUPPORT.md) - Know what works where
2. [Cross-Building Guide](./advanced/CROSS_BUILDING.md) - Multi-platform builds
3. [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md) - Platform-specific issues
4. [What NOT to Sync](./advanced/WHAT_NOT_TO_SYNC.md) - Multi-machine setup

**Time:** ~2 hours

## 📈 Documentation Stats

**New content added:**
- 6 new major guides (~3,500 lines)
- 4 significantly enhanced guides
- 200+ new code examples
- Complete cross-platform coverage

**Documentation coverage:**
- ✅ All major features documented
- ✅ All platforms covered
- ✅ Security best practices throughout
- ✅ Real-world use cases
- ✅ Troubleshooting for common issues

## 🆘 Getting Help

**Can't find what you need?**

1. **Search the docs:** Use your browser's search (Ctrl/Cmd+F)
2. **Check FAQ:** [FAQ.md](./FAQ.md)
3. **Check Quick Reference:** [QUICK_REFERENCE.md](./QUICK_REFERENCE.md)
4. **Run diagnostics:** `goenv doctor`
5. **Open an issue:** [GitHub Issues](https://github.com/go-nv/goenv/issues)
6. **Start a discussion:** [GitHub Discussions](https://github.com/go-nv/goenv/discussions)

## 🚀 What's Next

**Coming soon:**
- Video tutorials
- Interactive examples
- More platform-specific guides
- Community contributions

**Want to contribute?** See [CONTRIBUTING.md](./CONTRIBUTING.md#documentation-contributions)

## 📝 Feedback

We're always improving documentation. Your feedback helps!

- **Something unclear?** Open an issue
- **Found a typo?** Submit a quick PR
- **Want more examples?** Let us know what topics
- **Have suggestions?** Start a discussion

**Thank you for using goenv!** 🎉
