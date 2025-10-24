# Codebase Dump Script

## Overview

The `generate_codebase_dump.sh` script generates comprehensive dumps of the goenv codebase for analysis, documentation, and LLM consumption.

## Generated Files

The script generates **three separate files**:

### 1. `goenv_codebase.txt` (~2MB, ~50K lines)
**Complete source code dump**

Contains:
- Configuration files (go.mod, Makefile, etc.)
- Main package (main.go)
- All command files (cmd/*.go)
- All internal packages (internal/*/*.go)
- Repository statistics

Perfect for:
- LLM analysis (Claude, GPT, etc.)
- Code review and auditing
- Understanding implementation details
- Finding specific code patterns

### 2. `goenv_structure.txt` (~34KB, ~1K lines)
**Complete file structure and organization**

Contains:
- Full directory and file tree
- File statistics by directory (Go, Docs, Scripts, Other)
- Numbered file lists by type:
  - Go source files (*.go)
  - Documentation files (*.md)
  - Shell scripts (*.sh)
  - Configuration files
- Package organization overview

Perfect for:
- Understanding project structure
- Finding specific files
- Getting file counts and organization
- Navigating the codebase

### 3. `goenv_documentation.txt` (~355KB, ~13K lines)
**Complete documentation dump**

Contains:
- Table of contents
- All documentation files from docs/
- README.md from repository root
- Organized by directory structure

Perfect for:
- Understanding features and usage
- Reading all documentation in one place
- Searching across all docs
- LLM context about project purpose

## Usage

### Basic Usage
```bash
# Generate dumps in default location (/tmp/goenv_dumps/)
./scripts/generate_codebase_dump.sh
```

### Custom Output Directory
```bash
# Generate dumps in a specific directory
./scripts/generate_codebase_dump.sh ~/Desktop/goenv_analysis

# Generate dumps in current directory
./scripts/generate_codebase_dump.sh ./dumps
```

## Quick Commands

### View Files
```bash
# View codebase (source code)
cat /tmp/goenv_dumps/goenv_codebase.txt | less

# View structure (file tree)
cat /tmp/goenv_dumps/goenv_structure.txt | less

# View documentation
cat /tmp/goenv_dumps/goenv_documentation.txt | less
```

### Search Files
```bash
# Search for a pattern in codebase
grep -n "func.*Install" /tmp/goenv_dumps/goenv_codebase.txt

# Find file by name in structure
grep "doctor.go" /tmp/goenv_dumps/goenv_structure.txt

# Search documentation
grep -n "environment variable" /tmp/goenv_dumps/goenv_documentation.txt
```

### Copy Files
```bash
# Copy codebase to clipboard (macOS)
cat /tmp/goenv_dumps/goenv_codebase.txt | pbcopy

# Copy all dumps to clipboard (macOS)
cat /tmp/goenv_dumps/*.txt | pbcopy

# Copy to clipboard (Linux with xclip)
cat /tmp/goenv_dumps/goenv_codebase.txt | xclip -selection clipboard
```

## Example Workflows

### 1. LLM Analysis
```bash
# Generate fresh dumps
./scripts/generate_codebase_dump.sh

# Copy structure for overview
cat /tmp/goenv_dumps/goenv_structure.txt | pbcopy

# Then copy codebase for detailed analysis
cat /tmp/goenv_dumps/goenv_codebase.txt | pbcopy
```

### 2. Code Review
```bash
# Generate dumps in review directory
./scripts/generate_codebase_dump.sh ./code_review

# Review structure
cat ./code_review/goenv_structure.txt | less

# Search for specific patterns
grep -n "TODO\|FIXME\|XXX" ./code_review/goenv_codebase.txt
```

### 3. Documentation Review
```bash
# Generate dumps
./scripts/generate_codebase_dump.sh

# Review all documentation
cat /tmp/goenv_dumps/goenv_documentation.txt | less

# Find documentation about a feature
grep -n "cache" /tmp/goenv_dumps/goenv_documentation.txt
```

### 4. Project Analysis
```bash
# Generate dumps
./scripts/generate_codebase_dump.sh

# Get file statistics
tail -100 /tmp/goenv_dumps/goenv_structure.txt

# Count specific file types
grep "\.go$" /tmp/goenv_dumps/goenv_structure.txt | wc -l
```

## File Sizes

Typical file sizes (as of current version):

| File | Size | Lines | Description |
|------|------|-------|-------------|
| goenv_codebase.txt | ~2.0 MB | ~50K | All source code |
| goenv_structure.txt | ~34 KB | ~1K | File structure |
| goenv_documentation.txt | ~355 KB | ~13K | All documentation |

## Tips

1. **Start with structure**: Always review `goenv_structure.txt` first to understand the project layout

2. **Use grep for search**: Instead of reading entire files, use grep to find specific patterns:
   ```bash
   grep -n "pattern" file.txt
   ```

3. **Combine with LLMs**: The dumps are optimized for LLM consumption. Structure file provides context, codebase provides implementation details

4. **Regular regeneration**: Regenerate dumps after significant changes to keep them current

5. **Split for large LLMs**: If hitting token limits, provide structure + docs first, then specific code files

## Integration with LLMs

### Claude Code
```bash
# Generate dumps
./scripts/generate_codebase_dump.sh

# Use Read tool to read specific files:
# /tmp/goenv_dumps/goenv_structure.txt   (start here for overview)
# /tmp/goenv_dumps/goenv_codebase.txt    (for code analysis)
# /tmp/goenv_dumps/goenv_documentation.txt (for feature understanding)
```

### Other LLMs (ChatGPT, etc.)
```bash
# Copy structure for context
cat /tmp/goenv_dumps/goenv_structure.txt | pbcopy

# Then copy relevant sections of codebase
# (split if needed due to token limits)
```

## Maintenance

The script automatically excludes:
- `.git` directory
- `vendor` directory
- `node_modules` directory
- Test output files
- Hidden files in subdirectories

To modify exclusions, edit the `find` commands in the script.
