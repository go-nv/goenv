package meta

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanReplaceBinary(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to any directory")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "goenv")
	require.NoError(t, os.WriteFile(binary, []byte("x"), 0o755))

	t.Run("writable directory", func(t *testing.T) {
		assert.NoError(t, canReplaceBinary(binary))
	})

	// A read-only binary in a writable directory is still replaceable: the
	// update renames over it rather than writing through it.
	t.Run("read-only binary in writable directory", func(t *testing.T) {
		require.NoError(t, os.Chmod(binary, 0o444))
		t.Cleanup(func() { os.Chmod(binary, 0o755) })

		assert.NoError(t, canReplaceBinary(binary))
	})

	t.Run("read-only directory", func(t *testing.T) {
		// os.Chmod on Windows only toggles FILE_ATTRIBUTE_READONLY, which says
		// nothing about creating entries inside a directory. Real denials there
		// come from ACLs, which cannot be set portably from a test — so the
		// unwritable case is only reproducible on Unix. canReplaceBinary itself
		// is platform-agnostic: it probes with a real file, so an ACL-protected
		// directory like C:\Program Files fails it for real.
		if runtime.GOOS == "windows" {
			t.Skip("an unwritable directory cannot be simulated with os.Chmod on Windows")
		}

		require.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { os.Chmod(dir, 0o755) })

		assert.Error(t, canReplaceBinary(binary))
	})

	t.Run("leaves no probe file behind", func(t *testing.T) {
		require.NoError(t, canReplaceBinary(binary))

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1, "probe file should be removed")
	})
}

func TestPackageManager(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantManager string
		wantCommand string
	}{
		{
			name:        "homebrew on apple silicon",
			path:        "/opt/homebrew/Cellar/goenv/3.1.5/bin/goenv",
			wantManager: "Homebrew",
			wantCommand: "brew upgrade goenv",
		},
		{
			name:        "linuxbrew",
			path:        "/home/linuxbrew/.linuxbrew/bin/goenv",
			wantManager: "Homebrew",
			wantCommand: "brew upgrade goenv",
		},
		{
			name:        "nix store",
			path:        "/nix/store/abc123-goenv-3.1.5/bin/goenv",
			wantManager: "Nix",
			wantCommand: "nix profile upgrade goenv",
		},
		{
			name:        "scoop shim",
			path:        `C:\Users\dev\scoop\apps\goenv\current\goenv.exe`,
			wantManager: "Scoop",
			wantCommand: "scoop update goenv",
		},
		{
			name: "plain install is not managed",
			path: "/usr/local/bin/goenv",
		},
		{
			name: "user install is not managed",
			path: "/home/dev/.local/bin/goenv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, command := packageManager(tt.path)

			assert.Equal(t, tt.wantManager, manager)
			assert.Equal(t, tt.wantCommand, command)
		})
	}
}

func TestElevationInstructions(t *testing.T) {
	// filepath.Dir cleans to the host separator, and the advice itself branches
	// on the platform, so both sides of the assertion have to.
	binary, wantDir, wantHint := "/usr/local/bin/goenv", "/usr/local/bin", "sudo goenv update"
	if runtime.GOOS == "windows" {
		binary, wantDir, wantHint = `C:\Program Files\goenv\goenv.exe`, `C:\Program Files\goenv`, "Administrator"
	}

	instructions := elevationInstructions(binary)

	assert.Contains(t, instructions, wantDir)
	assert.Contains(t, instructions, wantHint)
	assert.Contains(t, instructions, "https://github.com/go-nv/goenv/releases")
}

func TestElevatedStartCommand_QuotesScriptPath(t *testing.T) {
	cmd := elevatedStartCommand(`C:\Users\Some User\AppData\Local\Temp\goenv-update.bat`)

	joined := cmd.Args[len(cmd.Args)-1]
	assert.Contains(t, joined, "-Verb RunAs")
	assert.Contains(t, joined, `'"C:\Users\Some User\AppData\Local\Temp\goenv-update.bat"'`,
		"a path with spaces must survive Start-Process argument splitting")
}
