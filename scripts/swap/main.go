package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Colors
const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	nc     = "\033[0m"
)

var (
	scriptDir   string
	goBinary    string
	backupDir   string
	goenvPath   string
	useColor    = true
	dryRun      bool
	updateAll   bool
	interactive = true
)

func init() {
	// Disable colors if not a TTY
	if !isatty() {
		useColor = false
	}

	// For `go run`, use the source file location
	// For built binary, use the executable location
	_, sourceFile, _, _ := runtime.Caller(0)
	scriptDir = filepath.Dir(sourceFile) // This gives us scripts/swap directory

	// Find repo root (go up from scripts/swap to repo root)
	repoRoot := filepath.Dir(filepath.Dir(scriptDir)) // Go up 2 levels

	goBinary = filepath.Join(repoRoot, "goenv")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}

	homeDir, _ := os.UserHomeDir()
	backupDir = filepath.Join(homeDir, ".goenv_backup")
}

func isatty() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func log(msg string) {
	if useColor {
		fmt.Printf("%s→%s %s\n", blue, nc, msg)
	} else {
		fmt.Printf("→ %s\n", msg)
	}
}

func success(msg string) {
	if useColor {
		fmt.Printf("%s✓%s %s\n", green, nc, msg)
	} else {
		fmt.Printf("✓ %s\n", msg)
	}
}

func warn(msg string) {
	if useColor {
		fmt.Printf("%s⚠%s %s\n", yellow, nc, msg)
	} else {
		fmt.Printf("⚠ %s\n", msg)
	}
}

func errorExit(msg string) {
	if useColor {
		fmt.Printf("%s✗%s %s\n", red, nc, msg)
	} else {
		fmt.Printf("✗ %s\n", msg)
	}
	os.Exit(1)
}

func detectGoenv() (string, error) {
	installations := detectAllGoenv()
	if len(installations) == 0 {
		return "", fmt.Errorf("goenv not found")
	}
	return installations[0], nil
}

func detectAllGoenv() []string {
	var found []string
	seen := make(map[string]bool)

	// Method 1: Check PATH
	if path, err := exec.LookPath("goenv"); err == nil {
		// Resolve symlinks to get actual file
		resolved := path
		if r, err := filepath.EvalSymlinks(path); err == nil {
			resolved = r
		}
		if !seen[resolved] {
			found = append(found, resolved)
			seen[resolved] = true
		}
	}

	// Build list of common locations to check
	homeDir, _ := os.UserHomeDir()
	locations := []string{}

	// Method 2: Homebrew locations
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		locations = append(locations,
			"/opt/homebrew/bin/goenv",              // ARM Mac
			"/usr/local/bin/goenv",                 // Intel Mac / Linux Homebrew
			"/home/linuxbrew/.linuxbrew/bin/goenv", // Linux Homebrew
		)
	}

	// Method 3: Manual installation
	locations = append(locations, filepath.Join(homeDir, ".goenv", "bin", "goenv"))

	// Method 4: System locations (Unix)
	if runtime.GOOS != "windows" {
		locations = append(locations,
			"/usr/bin/goenv",
			"/usr/local/bin/goenv",
			"/opt/goenv/bin/goenv",
		)
	}

	// Method 5: Windows locations
	if runtime.GOOS == "windows" {
		locations = append(locations,
			filepath.Join(homeDir, "bin", "goenv.exe"),
			filepath.Join(homeDir, ".goenv", "bin", "goenv.exe"),
			"C:\\Program Files\\goenv\\goenv.exe",
			"C:\\goenv\\bin\\goenv.exe",
		)

		// Check scoop
		if scoopPath := os.Getenv("SCOOP"); scoopPath != "" {
			locations = append(locations, filepath.Join(scoopPath, "shims", "goenv.exe"))
		}

		// Check chocolatey
		if programData := os.Getenv("ProgramData"); programData != "" {
			locations = append(locations, filepath.Join(programData, "chocolatey", "bin", "goenv.exe"))
		}
	}

	// Check all locations
	for _, loc := range locations {
		if stat, err := os.Stat(loc); err == nil && !stat.IsDir() {
			// Resolve symlinks
			resolved := loc
			if r, err := filepath.EvalSymlinks(loc); err == nil {
				resolved = r
			}

			if !seen[resolved] {
				found = append(found, resolved)
				seen[resolved] = true
			}
		}
	}

	return found
}

func detectShellOverrides() []string {
	var warnings []string

	// Check for shell function/alias (bash)
	shells := []string{"bash", "zsh"}
	for _, shell := range shells {
		if _, err := exec.LookPath(shell); err == nil {
			cmd := exec.Command(shell, "-c", "type -t goenv 2>/dev/null")
			if output, err := cmd.Output(); err == nil {
				outStr := string(output)
				if outStr == "function\n" {
					warnings = append(warnings, fmt.Sprintf("Shell function 'goenv' detected in %s", shell))
				} else if outStr == "alias\n" {
					warnings = append(warnings, fmt.Sprintf("Shell alias 'goenv' detected in %s", shell))
				}
			}
		}
	}

	return warnings
}

func checkGoenv() {
	var err error
	goenvPath, err = detectGoenv()
	if err != nil {
		errorExit(`goenv not found. Please install goenv first.

Options:
  - Homebrew:    brew install goenv
  - Manual:      git clone https://github.com/go-nv/goenv ~/.goenv
  - Package mgr: apt/yum/pkg install goenv`)
	}

	log(fmt.Sprintf("Found goenv: %s", goenvPath))
}

func cmdBuild() {
	log("Building Go version...")

	// Check if Go is installed
	if _, err := exec.LookPath("go"); err != nil {
		errorExit(`Go compiler not found. Please install Go first:
  - macOS:  brew install go
  - Linux:  apt install golang / yum install golang
  - Manual: https://golang.org/dl/`)
	}

	// Find repo root (where Makefile is located)
	repoRoot := scriptDir
	for i := 0; i < 3; i++ { // Go up max 3 levels
		makefilePath := filepath.Join(repoRoot, "Makefile")
		if _, err := os.Stat(makefilePath); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}

	log(fmt.Sprintf("Running: make build (in %s)", repoRoot))
	cmd := exec.Command("make", "build")
	cmd.Dir = repoRoot // Set working directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errorExit("Build failed")
	}

	if _, err := os.Stat(goBinary); os.IsNotExist(err) {
		errorExit(fmt.Sprintf("Build completed but binary not found: %s", goBinary))
	}

	success(fmt.Sprintf("Built: %s", goBinary))

	// Show version
	cmd = exec.Command(goBinary, "--version")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func cmdStatus() {
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  goenv Status")
	fmt.Println("═══════════════════════════════════════")

	log(fmt.Sprintf("System: %s %s", runtime.GOOS, runtime.GOARCH))
	fmt.Println()

	installations := detectAllGoenv()
	if len(installations) == 0 {
		warn("goenv not found in PATH or common locations")
	} else if len(installations) == 1 {
		log(fmt.Sprintf("goenv location: %s", installations[0]))
		showGoenvInfo(installations[0])
	} else {
		warn(fmt.Sprintf("Found %d goenv installations:", len(installations)))
		for i, path := range installations {
			fmt.Printf("\n  %d. %s\n", i+1, path)
			showGoenvInfo(path)
		}
		fmt.Println()
		warn("Multiple installations may cause conflicts!")
		warn("Use 'goenv doctor' to check for issues")
	}

	// Check for shell overrides
	if overrides := detectShellOverrides(); len(overrides) > 0 {
		fmt.Println()
		for _, override := range overrides {
			warn(override)
		}
	}

	fmt.Println()
	log(fmt.Sprintf("Go binary: %s", goBinary))
	if stat, err := os.Stat(goBinary); err == nil {
		success(fmt.Sprintf("Exists (size: %d bytes)", stat.Size()))
	} else {
		warn("Not built yet (run: swap build)")
	}

	fmt.Println()
	log(fmt.Sprintf("Backup: %s", backupDir))
	backupFile := filepath.Join(backupDir, "goenv.bash")
	if _, err := os.Stat(backupFile); err == nil {
		success("Exists")
	} else {
		warn("No backup (will create on first swap)")
	}
	fmt.Println("═══════════════════════════════════════")
}

func showGoenvInfo(path string) {
	if fileInfo, err := os.Stat(path); err == nil {
		// Check if it's a binary or script
		if fileInfo.Mode()&0111 != 0 {
			// Read first bytes to determine type
			f, err := os.Open(path)
			if err == nil {
				buf := make([]byte, 4)
				f.Read(buf)
				f.Close()

				// Check for ELF (Linux) or Mach-O (macOS) magic numbers
				if buf[0] == 0x7f && buf[1] == 0x45 && buf[2] == 0x4c && buf[3] == 0x46 {
					fmt.Printf("     Type: Go version (ELF binary)\n")
				} else if buf[0] == 0xcf && buf[1] == 0xfa {
					fmt.Printf("     Type: Go version (Mach-O binary)\n")
				} else if buf[0] == 0x4d && buf[1] == 0x5a {
					fmt.Printf("     Type: Go version (PE binary)\n")
				} else if buf[0] == '#' && buf[1] == '!' {
					fmt.Printf("     Type: Bash version (script)\n")
				} else {
					fmt.Printf("     Type: Unknown\n")
				}
			}
		}

		// Show version
		cmd := exec.Command(path, "--version")
		if output, err := cmd.Output(); err == nil {
			fmt.Printf("     Version: %s", string(output))
		}
	}
}

func cmdGo() {
	log("Switching to Go version...")

	installations := detectAllGoenv()
	if len(installations) == 0 {
		errorExit(`goenv not found. Please install goenv first.

Options:
  - Homebrew:    brew install goenv
  - Manual:      git clone https://github.com/go-nv/goenv ~/.goenv
  - Package mgr: apt/yum/pkg install goenv`)
	}

	// Check if Go binary exists
	if _, err := os.Stat(goBinary); os.IsNotExist(err) {
		warn("Go binary not built. Building now...")
		cmdBuild()
	}

	// Determine which installations to update
	targets := []string{}
	if updateAll {
		targets = installations
		log(fmt.Sprintf("Updating all %d installations...", len(installations)))
	} else if len(installations) > 1 && interactive && !dryRun {
		// Interactive selection
		fmt.Println()
		warn(fmt.Sprintf("Found %d goenv installations:", len(installations)))
		for i, path := range installations {
			fmt.Printf("  %d. %s\n", i+1, path)
		}
		fmt.Println()
		fmt.Print("Which installation do you want to update? [1, or 'all']: ")

		var choice string
		fmt.Scanln(&choice)

		if choice == "all" || choice == "a" {
			targets = installations
		} else if choice == "" || choice == "1" {
			targets = []string{installations[0]}
		} else {
			// Parse number
			var num int
			fmt.Sscanf(choice, "%d", &num)
			if num > 0 && num <= len(installations) {
				targets = []string{installations[num-1]}
			} else {
				errorExit("Invalid selection")
			}
		}
	} else {
		targets = []string{installations[0]}
	}

	// Update each target
	for _, target := range targets {
		fmt.Println()
		log(fmt.Sprintf("Updating: %s", target))

		if dryRun {
			success(fmt.Sprintf("[DRY RUN] Would update: %s", target))
			continue
		}

		swapGoenvBinary(target)
	}

	fmt.Println()
	success("Switch successful!")
	warn("IMPORTANT: Reload your shell before testing:")
	fmt.Println("  hash -r")
	fmt.Println("  # OR restart your terminal")
	fmt.Println()
	warn("To test: goenv --version")
	warn("If it hangs, swap back with: ./swap bash")
}

func swapGoenvBinary(target string) {
	// Create backup if it doesn't exist
	backupFile := filepath.Join(backupDir, filepath.Base(target)+".bash")
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		log("Creating backup...")
		os.MkdirAll(backupDir, 0755)

		src, err := os.Open(target)
		if err != nil {
			errorExit(fmt.Sprintf("Cannot read goenv: %v", err))
		}
		defer src.Close()

		dst, err := os.Create(backupFile)
		if err != nil {
			errorExit(fmt.Sprintf("Cannot create backup: %v", err))
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			errorExit(fmt.Sprintf("Backup failed: %v", err))
		}

		success(fmt.Sprintf("Backed up: %s", backupFile))
	}

	// Copy Go binary to goenv location
	log("Replacing with Go version...")

	src, err := os.Open(goBinary)
	if err != nil {
		errorExit(fmt.Sprintf("Cannot read Go binary: %v", err))
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		// Try with sudo if regular copy fails (Unix only)
		if runtime.GOOS != "windows" {
			log("Regular copy failed, trying with sudo...")
			cmd := exec.Command("sudo", "cp", goBinary, target)
			if err := cmd.Run(); err == nil {
				success(fmt.Sprintf("Copied: %s → %s (with sudo)", goBinary, target))
			} else {
				errorExit(fmt.Sprintf("Cannot copy to %s\n\nTry manually:\n  sudo cp %s %s", target, goBinary, target))
			}
		} else {
			errorExit(fmt.Sprintf("Cannot write to %s: %v", target, err))
		}
	} else {
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			errorExit(fmt.Sprintf("Copy failed: %v", err))
		}
		dst.Chmod(0755)
		success(fmt.Sprintf("Copied: %s → %s", goBinary, target))

		// Make executable (Unix only - Windows uses file extension)
		if runtime.GOOS != "windows" {
			if err := os.Chmod(target, 0755); err != nil {
				warn(fmt.Sprintf("Could not set executable permission: %v", err))
			}
		}
	}

	// Verify file was copied
	if stat, err := os.Stat(target); err == nil {
		success(fmt.Sprintf("Binary installed (%d bytes)", stat.Size()))
	} else {
		errorExit(fmt.Sprintf("Verification failed: %v", err))
	}
}

func cmdBash() {
	log("Switching back to bash version...")

	checkGoenv()

	backupFile := filepath.Join(backupDir, "goenv.bash")

	// Check if backup exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		warn("No backup found.")

		// Try reinstalling from package manager
		if runtime.GOOS == "darwin" {
			if _, err := exec.LookPath("brew"); err == nil {
				log("Reinstalling from Homebrew...")
				cmd := exec.Command("brew", "reinstall", "goenv")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err == nil {
					success("Reinstalled from Homebrew")
					return
				}
			}
		}

		errorExit("Cannot restore: No backup found")
	}

	// Restore from backup
	log("Restoring from backup...")

	src, err := os.Open(backupFile)
	if err != nil {
		errorExit(fmt.Sprintf("Cannot read backup: %v", err))
	}
	defer src.Close()

	dst, err := os.Create(goenvPath)
	if err != nil {
		// Try with sudo
		if runtime.GOOS != "windows" {
			log("Regular copy failed, trying with sudo...")
			cmd := exec.Command("sudo", "cp", backupFile, goenvPath)
			if err := cmd.Run(); err == nil {
				success(fmt.Sprintf("Restored: %s → %s (with sudo)", backupFile, goenvPath))
			} else {
				errorExit(fmt.Sprintf("Cannot restore to %s", goenvPath))
			}
		} else {
			errorExit(fmt.Sprintf("Cannot write to %s: %v", goenvPath, err))
		}
	} else {
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			errorExit(fmt.Sprintf("Restore failed: %v", err))
		}
		dst.Chmod(0755)
		success(fmt.Sprintf("Restored: %s → %s", backupFile, goenvPath))
	}

	success("Switch successful!")
	warn("Reload your shell: hash -r (or restart terminal)")
}

func printUsage() {
	fmt.Println("Usage: swap {build|go|bash|status} [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build   - Build the Go version")
	fmt.Println("  go      - Switch to Go version")
	fmt.Println("  bash    - Switch back to bash version")
	fmt.Println("  status  - Show current version and status")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --all       Update all goenv installations (use with 'go')")
	fmt.Println("  --dry-run   Show what would be done without actually doing it")
	fmt.Println("  --yes       Non-interactive mode, use default selection")
	fmt.Println()
	fmt.Println("Cross-platform: Works on macOS, Linux, BSD, WSL, Windows")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  swap build              # Build Go version first")
	fmt.Println("  swap status             # Check current version")
	fmt.Println("  swap go                 # Switch to Go version (interactive)")
	fmt.Println("  swap go --all           # Update all installations")
	fmt.Println("  swap go --dry-run       # Preview changes without applying")
	fmt.Println("  swap go --yes           # Non-interactive, update first found")
	fmt.Println("  swap bash               # Switch back to bash version")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse flags
	cmd := os.Args[1]
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--all", "-a":
			updateAll = true
		case "--dry-run", "-n":
			dryRun = true
		case "--yes", "-y":
			interactive = false
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			errorExit(fmt.Sprintf("Unknown flag: %s", os.Args[i]))
		}
	}

	switch cmd {
	case "build":
		cmdBuild()
	case "go":
		if useColor {
			fmt.Printf("%s╔═══════════════════════════════════════════╗%s\n", green, nc)
			fmt.Printf("%s║     Switching to Go version of goenv      ║%s\n", green, nc)
			fmt.Printf("%s╚═══════════════════════════════════════════╝%s\n", green, nc)
		} else {
			fmt.Println("Switching to Go version of goenv")
		}
		fmt.Println()
		if dryRun {
			log("[DRY RUN MODE] - No changes will be made")
			fmt.Println()
		}
		cmdGo()
	case "bash":
		if useColor {
			fmt.Printf("%s╔═══════════════════════════════════════════╗%s\n", yellow, nc)
			fmt.Printf("%s║    Switching to Bash version of goenv     ║%s\n", yellow, nc)
			fmt.Printf("%s╚═══════════════════════════════════════════╝%s\n", yellow, nc)
		} else {
			fmt.Println("Switching to Bash version of goenv")
		}
		fmt.Println()
		cmdBash()
	case "status":
		cmdStatus()
	default:
		printUsage()
		os.Exit(1)
	}
}
