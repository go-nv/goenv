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
	scriptDir string
	goBinary  string
	backupDir string
	goenvPath string
	useColor  = true
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
	// Method 1: Check PATH
	path, err := exec.LookPath("goenv")
	if err == nil {
		// Resolve symlinks
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			return resolved, nil
		}
		return path, nil
	}

	// Method 2: Check Homebrew
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		brewPaths := []string{
			"/opt/homebrew/bin/goenv",              // ARM Mac
			"/usr/local/bin/goenv",                 // Intel Mac / Linux Homebrew
			"/home/linuxbrew/.linuxbrew/bin/goenv", // Linux Homebrew
		}

		for _, p := range brewPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	// Method 3: Check manual installation
	homeDir, _ := os.UserHomeDir()
	manualPath := filepath.Join(homeDir, ".goenv", "bin", "goenv")
	if _, err := os.Stat(manualPath); err == nil {
		return manualPath, nil
	}

	// Method 4: Check system (Unix only)
	if runtime.GOOS != "windows" {
		if _, err := os.Stat("/usr/bin/goenv"); err == nil {
			return "/usr/bin/goenv", nil
		}
	}

	return "", fmt.Errorf("goenv not found")
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

	goenvPath, err := detectGoenv()

	log(fmt.Sprintf("System: %s %s", runtime.GOOS, runtime.GOARCH))

	if err == nil {
		log(fmt.Sprintf("goenv location: %s", goenvPath))

		if fileInfo, err := os.Stat(goenvPath); err == nil {
			// Check if it's a binary or script
			if fileInfo.Mode()&0111 != 0 {
				// Read first bytes to determine type
				f, err := os.Open(goenvPath)
				if err == nil {
					buf := make([]byte, 4)
					f.Read(buf)
					f.Close()

					// Check for ELF (Linux) or Mach-O (macOS) magic numbers
					if buf[0] == 0x7f && buf[1] == 0x45 && buf[2] == 0x4c && buf[3] == 0x46 {
						success("Currently: Go version (ELF binary)")
					} else if buf[0] == 0xcf && buf[1] == 0xfa {
						success("Currently: Go version (Mach-O binary)")
					} else if buf[0] == '#' && buf[1] == '!' {
						warn("Currently: Bash version (script)")
					} else {
						fmt.Printf("  Type: Unknown\n")
					}
				}
			}

			log("Version:")
			cmd := exec.Command(goenvPath, "--version")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	} else {
		warn("goenv not found in PATH or common locations")
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

func cmdGo() {
	log("Switching to Go version...")

	checkGoenv()

	// Check if Go binary exists
	if _, err := os.Stat(goBinary); os.IsNotExist(err) {
		warn("Go binary not built. Building now...")
		cmdBuild()
	}

	// Create backup if it doesn't exist
	backupFile := filepath.Join(backupDir, "goenv.bash")
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		log("Creating backup...")
		os.MkdirAll(backupDir, 0755)

		src, err := os.Open(goenvPath)
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

	dst, err := os.Create(goenvPath)
	if err != nil {
		// Try with sudo if regular copy fails (Unix only)
		if runtime.GOOS != "windows" {
			log("Regular copy failed, trying with sudo...")
			cmd := exec.Command("sudo", "cp", goBinary, goenvPath)
			if err := cmd.Run(); err == nil {
				success(fmt.Sprintf("Copied: %s → %s (with sudo)", goBinary, goenvPath))
			} else {
				errorExit(fmt.Sprintf("Cannot copy to %s\n\nTry manually:\n  sudo cp %s %s", goenvPath, goBinary, goenvPath))
			}
		} else {
			errorExit(fmt.Sprintf("Cannot write to %s: %v", goenvPath, err))
		}
	} else {
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			errorExit(fmt.Sprintf("Copy failed: %v", err))
		}
		dst.Chmod(0755)
		success(fmt.Sprintf("Copied: %s → %s", goBinary, goenvPath))

		// Make executable
		if err := os.Chmod(goenvPath, 0755); err != nil {
			warn(fmt.Sprintf("Could not set executable permission: %v", err))
		}
	}
	
	// Verify file was copied
	if stat, err := os.Stat(goenvPath); err == nil {
		success(fmt.Sprintf("Binary installed (%d bytes)", stat.Size()))
	} else {
		errorExit(fmt.Sprintf("Verification failed: %v", err))
	}
	
	success("Switch successful!")
	warn("IMPORTANT: Reload your shell before testing:")
	fmt.Println("  hash -r")
	fmt.Println("  # OR restart your terminal")
	fmt.Println()
	warn("To test: goenv --version")
	warn("If it hangs, swap back with: ./swap bash")
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
	fmt.Println("Usage: swap {build|go|bash|status}")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build   - Build the Go version")
	fmt.Println("  go      - Switch to Go version")
	fmt.Println("  bash    - Switch back to bash version")
	fmt.Println("  status  - Show current version and status")
	fmt.Println()
	fmt.Println("Cross-platform: Works on macOS, Linux, BSD, WSL, Windows")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  swap build    # Build Go version first")
	fmt.Println("  swap status   # Check current version")
	fmt.Println("  swap go       # Switch to Go version")
	fmt.Println("  swap bash     # Switch back to bash version")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
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
