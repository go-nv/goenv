# Codebase Analysis Dump Scripts

This directory contains scripts for generating comprehensive codebase dumps suitable for analysis with LLMs or other tools.

## Scripts

### 1. `generate_codebase_dump.sh`

Generates a complete, structured dump of the entire goenv codebase.

**What's Included:**
- Directory structure
- Configuration files (go.mod, Makefile, etc.)
- Main package (main.go)
- All command files (cmd/*.go)
- All internal packages (internal/*/*.go)
- Documentation listing (without full content to keep size manageable)
- Repository statistics and package breakdown

**Usage:**
```bash
# Generate to default location (/tmp/goenv_codebase_analysis.txt)
./scripts/generate_codebase_dump.sh

# Generate to specific location
./scripts/generate_codebase_dump.sh ~/Desktop/goenv_dump.txt

# Generate and view
./scripts/generate_codebase_dump.sh /tmp/dump.txt && cat /tmp/dump.txt | less
```

**Output Size:** ~2 MB, ~48,000 lines

**Use Cases:**
- LLM analysis (Claude, GPT, etc.)
- Code review and auditing
- Understanding project structure
- Documentation generation
- Training code comprehension models

---

### 2. `generate_cmd_dump.sh`

Generates a concatenated dump of just the command files (cmd/*.go).

**What's Included:**
- File index with line counts
- All Go files in cmd/ directory (both implementation and test files)
- Clear separators between files

**Usage:**
```bash
# Generate to default location (/tmp/goenv_cmd_files.txt)
./scripts/generate_cmd_dump.sh

# Generate to specific location
./scripts/generate_cmd_dump.sh ~/Desktop/cmd_analysis.txt
```

**Output Size:** ~1 MB, ~25,000 lines

**Use Cases:**
- Analyzing command structure
- Command implementation review
- CLI interface analysis
- Focused code review of commands only

---

## Output Format

Both scripts generate well-structured text files with:

✅ **Clear headers** for each section
✅ **File separators** with filename and line count
✅ **Statistics** at the end
✅ **Easy to parse** format for automated tools
✅ **Human-readable** with proper formatting

### Example Output Structure:

```
################################################################################
# GOENV CODEBASE COMPLETE ANALYSIS DUMP
# Generated: [timestamp]
# Repository: [path]
################################################################################

# TABLE OF CONTENTS
1. Repository Structure
2. Configuration Files
...

################################################################################
# 1. REPOSITORY STRUCTURE
################################################################################
[directory tree]

─────────────────────────────────────────────────────────────────────────────
FILE: cmd/cache.go
LINES: 1234
─────────────────────────────────────────────────────────────────────────────
[file contents]

################################################################################
# 7. STATISTICS
################################################################################
[statistics and package breakdown]
```

---

## Quick Commands

After generating dumps:

```bash
# View the dump
cat /tmp/goenv_codebase_analysis.txt | less

# Copy to clipboard (macOS)
cat /tmp/goenv_codebase_analysis.txt | pbcopy

# Search for specific code
grep -n "cache clean" /tmp/goenv_codebase_analysis.txt

# Count functions
grep -c "^func " /tmp/goenv_codebase_analysis.txt

# Extract specific file
sed -n '/^FILE: cmd\/cache.go$/,/^FILE: /p' /tmp/goenv_codebase_analysis.txt | head -n -1
```

---

## Notes

- Scripts are designed to be portable (no external dependencies beyond standard Unix tools)
- Documentation files are listed but not included in full dump to keep size manageable
- If you need documentation included, you can run: `find docs -name '*.md' | sort | xargs cat >> output.txt`
- Scripts automatically exclude `.git`, `vendor`, and `node_modules` directories
- Output files are created in `/tmp` by default (won't clutter the repository)

---

## Maintenance

These scripts are located in the `/scripts` directory and should be updated when:
- New major directories are added to the repository
- The structure of important files changes
- Additional analysis capabilities are needed

To regenerate after code changes:
```bash
cd /path/to/goenv
./scripts/generate_codebase_dump.sh
```

The dumps will reflect the current state of the codebase at the time of generation.
