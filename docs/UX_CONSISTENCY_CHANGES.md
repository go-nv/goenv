# UX Consistency Changes

This document outlines all the changes needed to normalize command/UX inconsistencies across goenv.

## 1. JSON Output Coverage

### Issue
Some commands have `--json` flags while others don't, creating inconsistent automation support.

**Current state:**
- ✅ `cache status --json` - exists
- ✅ `inventory --json` - exists
- ✅ `vscode ... --json` - exists
- ✅ `doctor --json` - **IMPLEMENTED** (supports JSON output with exit codes)
- ✅ `tools list --json` - exists
- ✅ `versions --json` - **IMPLEMENTED**
- ✅ `list --json` - **IMPLEMENTED** (modern unified command, replaces `versions`)

---

### Change 1.1: Add `--json` to `doctor` command

**File:** `cmd/doctor.go`

**Add import:**
```go
import (
    "encoding/json"
    // ... other imports
)
```

**Update checkResult struct to support JSON:**
```go
type checkResult struct {
    Name    string `json:"name"`
    Status  string `json:"status"` // "ok", "warning", "error"
    Message string `json:"message"`
    Advice  string `json:"advice,omitempty"`
}
```

**Add flag variable:**
```go
var doctorJSON bool
```

**Update init():**
```go
func init() {
    rootCmd.AddCommand(doctorCmd)
    doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results in JSON format")
    helptext.SetCommandHelp(doctorCmd)
}
```

**Update runDoctor() to support JSON output:**
```go
func runDoctor(cmd *cobra.Command, args []string) error {
    cfg := config.Load()
    results := []checkResult{}

    // ... (all existing checks remain the same, just collect results)

    // At the end, replace the output section with:
    if doctorJSON {
        // JSON output
        type jsonOutput struct {
            Checks  []checkResult `json:"checks"`
            Summary struct {
                Total    int `json:"total"`
                OK       int `json:"ok"`
                Warnings int `json:"warnings"`
                Errors   int `json:"errors"`
            } `json:"summary"`
        }

        output := jsonOutput{Checks: results}
        for _, result := range results {
            output.Summary.Total++
            switch result.Status {
            case "ok":
                output.Summary.OK++
            case "warning":
                output.Summary.Warnings++
            case "error":
                output.Summary.Errors++
            }
        }

        encoder := json.NewEncoder(cmd.OutOrStdout())
        encoder.SetIndent("", "  ")
        return encoder.Encode(output)
    }

    // Existing human-readable output
    // ... (keep all existing fmt.Fprintln calls)
}
```

**Update Long description to mention flag:**
```go
Long: `Checks your goenv installation and configuration for common issues.

... (existing text) ...

Flags:
  --json    Output results in JSON format for CI/automation`,
```

---

### Change 1.2: Add `--json` to `tools list` command

**File:** `cmd/tools.go`

The `tools list` subcommand doesn't exist yet! Need to create it or it's handled elsewhere.

**If adding to existing tools command structure:**

```go
type toolInfo struct {
    Name    string `json:"name"`
    Version string `json:"version,omitempty"`
    Path    string `json:"path"`
    Host    string `json:"host"` // e.g. "darwin-arm64"
}

var toolsListJSON bool

// In tools list command init:
toolsListCmd.Flags().BoolVar(&toolsListJSON, "json", false, "Output in JSON format")

// In tools list RunE:
if toolsListJSON {
    encoder := json.NewEncoder(cmd.OutOrStdout())
    encoder.SetIndent("", "  ")
    return encoder.Encode(toolInfos)
}
```

---

### Change 1.3: Add `--json` to `versions` command

**File:** `cmd/versions.go`

**Add flag and JSON support:**
```go
var versionsJSON bool

func init() {
    rootCmd.AddCommand(versionsCmd)
    versionsCmd.Flags().BoolVar(&versionsJSON, "json", false, "Output in JSON format")
    helptext.SetCommandHelp(versionsCmd)
}

// In runVersions:
if versionsJSON {
    type versionInfo struct {
        Version   string `json:"version"`
        IsActive  bool   `json:"active"`
        Source    string `json:"source,omitempty"` // where it was set from
    }

    var versions []versionInfo
    // ... populate versions array

    encoder := json.NewEncoder(cmd.OutOrStdout())
    encoder.SetIndent("", "  ")
    return encoder.Encode(versions)
}
```

---

## 2. Emojis and Color in CLI Output

### Issue
Emojis are pretty but can break CI/log parsers and aren't appropriate when stdout isn't a TTY.

### Changes Needed:

#### Change 2.1: Add global --plain flag support

**File:** `cmd/root.go`

**Add global flags:**
```go
var (
    noColor bool
    plain   bool
)

func init() {
    rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
    rootCmd.PersistentFlags().BoolVar(&plain, "plain", false, "Plain output (no colors, no emojis)")
}
```

#### Change 2.2: Respect NO_COLOR environment variable

**File:** `internal/utils/output.go` (create new file)

```go
package utils

import (
    "os"

    "golang.org/x/term"
)

// ShouldUseEmojis returns true if emojis should be used in output
func ShouldUseEmojis() bool {
    // Respect NO_COLOR environment variable (https://no-color.org/)
    if os.Getenv("NO_COLOR") != "" {
        return false
    }

    // Check if --plain or --no-color flags are set
    // (these would need to be passed down or checked globally)

    // Check if stdout is a TTY
    if !term.IsTerminal(int(os.Stdout.Fd())) {
        return false
    }

    return true
}

// ShouldUseColor returns true if colored output should be used
func ShouldUseColor() bool {
    if os.Getenv("NO_COLOR") != "" {
        return false
    }

    if !term.IsTerminal(int(os.Stdout.Fd())) {
        return false
    }

    return true
}
```

#### Change 2.3: Update all emoji usage

**Files to update:** All cmd/*.go files with emojis

**Pattern to follow:**
```go
// OLD:
fmt.Fprintln(cmd.OutOrStdout(), "🔍 Checking goenv installation...")

// NEW:
if utils.ShouldUseEmojis() {
    fmt.Fprintln(cmd.OutOrStdout(), "🔍 Checking goenv installation...")
} else {
    fmt.Fprintln(cmd.OutOrStdout(), "Checking goenv installation...")
}
```

**Or use helper function:**
```go
func emoji(e string) string {
    if utils.ShouldUseEmojis() {
        return e + " "
    }
    return ""
}

// Usage:
fmt.Fprintf(cmd.OutOrStdout(), "%sChecking goenv installation...\n", emoji("🔍"))
```

**Commands with emojis to update:**
- `cmd/cache.go` - 📊 🔨 📦 ✓ ✗ 🔍 💡 ✅ 💾 ❌
- `cmd/doctor.go` - 🔍 ✓ ✗ ⚠️
- `cmd/install.go` - ⬇️ ✓ ✗
- `cmd/inventory.go` - 📦
- `cmd/vscode.go` - ✓ ✨
- Others as found

---

## 3. Prompts in CI

### Issue
`cache clean` requires interactive confirmation unless `--force`, which can hang CI pipelines.

#### Change 3.1: Make confirmation behavior clear in errors

**File:** `cmd/cache.go`

**Current prompt code is OK, but ensure error messages are clear:**

```go
// When no --force and not a TTY:
if !cleanForce && !term.IsTerminal(int(os.Stdin.Fd())) {
    return fmt.Errorf("refusing to clean caches non-interactively without --force flag\n" +
        "Use: goenv cache clean --force")
}
```

#### Change 3.2: Update ci-setup documentation

**File:** `cmd/ci-setup.go`

**Update the output/docs to mention:**
```go
fmt.Fprintln(cmd.OutOrStdout(), "Cache management in CI:")
fmt.Fprintln(cmd.OutOrStdout(), "  goenv cache clean --force          # Clean without prompts")
fmt.Fprintln(cmd.OutOrStdout(), "  goenv cache clean build --force    # Clean build cache only")
```

**File:** `docs/CI_CD_GUIDE.md`

**Add section:**
```markdown
### Cache Cleaning

When cleaning caches in CI, always use `--force` to skip interactive prompts:

```bash
# Clean build caches (safe for CI)
goenv cache clean build --force

# Clean all caches
goenv cache clean all --force
```

Without `--force`, the command will error in non-interactive environments.
```

---

## Implementation Checklist

- [ ] Add `--json` to `doctor` command
- [ ] Add `--json` to `tools list` command (or create if missing)
- [ ] Add `--json` to `versions` command
- [ ] Create `internal/utils/output.go` with TTY detection
- [ ] Add `--plain` and `--no-color` global flags
- [ ] Update all emoji usage to be TTY-aware in:
  - [ ] `cmd/cache.go`
  - [ ] `cmd/doctor.go`
  - [ ] `cmd/install.go`
  - [ ] `cmd/inventory.go`
  - [ ] `cmd/vscode.go`
  - [ ] Other commands as discovered
- [ ] Update `cache clean` to detect non-TTY and require `--force`
- [ ] Update `ci-setup` output to mention `--force`
- [ ] Update `docs/CI_CD_GUIDE.md` with cache cleaning guidance

---

## Testing

After implementation:

```bash
# Test JSON outputs
goenv doctor --json | jq .
goenv versions --json | jq .
goenv cache status --json | jq .

# Test NO_COLOR
NO_COLOR=1 goenv doctor
goenv doctor --plain

# Test non-TTY behavior
goenv doctor | cat  # Should have no emojis
echo | goenv cache clean  # Should error without --force

# Test --force in non-interactive
echo | goenv cache clean --force  # Should work
```

---

## Migration Notes

- The `checkResult` struct change is backward compatible (adds JSON tags)
- Emoji changes are backward compatible (behavior only changes based on TTY/env)
- `--force` requirement is a breaking change for CI scripts, but safer default
- Document all flags in `--help` output

---

## Priority Order

**High Priority** (affects automation/CI):
1. Add `--json` flags to all commands
2. Fix `--force` requirement detection

**Medium Priority** (quality of life):
3. Add NO_COLOR support
4. Add global `--plain` flag

**Low Priority** (polish):
5. Make all emojis TTY-aware
