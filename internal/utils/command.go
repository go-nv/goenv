package utils

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

// RunCommand executes a command and returns the error (no output capture).
// This is useful when you only care about whether the command succeeded,
// not its output.
//
// Example:
//
//	if err := RunCommand("git", "add", "."); err != nil {
//	    return errors.FailedTo("stage changes", err)
//	}
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// RunCommandInDir executes a command in a specific directory and returns the error.
// This is useful for running commands in a different working directory without
// changing the process's current directory.
//
// Example:
//
//	if err := RunCommandInDir("/path/to/repo", "git", "pull"); err != nil {
//	    return errors.FailedTo("pull updates", err)
//	}
func RunCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

// RunCommandOutputInDir executes a command in a specific directory and returns
// its trimmed output. This combines the directory setting with output capture.
//
// Example:
//
//	commit, err := RunCommandOutputInDir("/path/to/repo", "git", "rev-parse", "HEAD")
//	if err != nil {
//	    return "", errors.FailedTo("get commit", err)
//	}
func RunCommandOutputInDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandWithIO executes a command with custom stdout and stderr writers.
// This is useful when you want to redirect command output to specific destinations,
// such as the terminal or a file.
//
// Example:
//
//	if err := RunCommandWithIO("git", []string{"pull"}, os.Stdout, os.Stderr); err != nil {
//	    return errors.FailedTo("pull updates", err)
//	}
func RunCommandWithIO(name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// RunCommandWithIOInDir executes a command in a specific directory with custom
// stdout and stderr writers. This combines directory setting with I/O redirection.
//
// Example:
//
//	err := RunCommandWithIOInDir("/path/to/repo", "git", []string{"pull"}, os.Stdout, os.Stderr)
//	if err != nil {
//	    return errors.FailedTo("pull updates", err)
//	}
func RunCommandWithIOInDir(dir, name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// RunCommandOutput executes a command and returns its trimmed output.
// This is a common pattern throughout the codebase to run commands and
// get their output as a clean string (no leading/trailing whitespace).
//
// Example:
//
//	output, err := RunCommandOutput("go", "version")
//	if err != nil {
//	    return err
//	}
//	fmt.Println(output) // "go version go1.21.0 darwin/arm64"
func RunCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandOutputContext is like RunCommandOutput but accepts a context for
// cancellation and timeout control.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	output, err := RunCommandOutputContext(ctx, "go", "version")
func RunCommandOutputContext(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandCombinedOutput executes a command and returns its combined stdout
// and stderr output as a trimmed string. This is useful when the command might
// write output to stderr that you need to capture (e.g., compiler version info).
//
// Example:
//
//	output, err := RunCommandCombinedOutput("clang", "-v")
//	if err != nil {
//	    return err
//	}
//	fmt.Println(output) // clang version output (often goes to stderr)
func RunCommandCombinedOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandLines executes a command and returns its output as a slice of
// non-empty, trimmed lines.
//
// Example:
//
//	lines, err := RunCommandLines("go", "list", "-m", "all")
//	for _, line := range lines {
//	    fmt.Println(line)
//	}
func RunCommandLines(name string, args ...string) ([]string, error) {
	output, err := RunCommandOutput(name, args...)
	if err != nil {
		return nil, err
	}
	return SplitLines(output), nil
}
