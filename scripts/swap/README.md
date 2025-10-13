# goenv Swap Utility

A cross-platform utility to swap between bash and Go versions of goenv for testing.

## Building

```bash
cd scripts/swap
go build -o swap main.go
```

Or from the project root:

```bash
go build -o scripts/swap/swap scripts/swap/main.go
```

## Usage

```bash
# Build the Go version of goenv
./swap build

# Check current status
./swap status

# Switch to Go version
./swap go

# Switch back to bash version
./swap bash
```

## What It Does

- **build**: Compiles the Go version of goenv
- **go**: Backs up the current bash goenv and replaces it with the Go version
- **bash**: Restores the bash version from backup
- **status**: Shows which version is currently active

## Platform Support

Works on:
- macOS (Intel and ARM)
- Linux
- BSD
- WSL
- Windows

## Notes

This is a **testing utility** for developers working on the goenv Go migration. It allows quick switching between implementations to compare behavior and test compatibility.

The utility automatically:
- Detects your goenv installation (Homebrew, manual, system)
- Creates backups before swapping
- Handles permissions (prompts for sudo if needed)
- Verifies the swap was successful
