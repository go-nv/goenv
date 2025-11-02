package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
)

func TestCopyFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Create a source file with specific permissions
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content\n")

	// On Unix, create with specific permissions; on Windows, use default
	var expectedMode os.FileMode
	if IsWindows() {
		// Windows doesn't support Unix-style permissions
		expectedMode = 0666
		testutil.WriteTestFile(t, srcPath, content, expectedMode, "Failed to create source file")
	} else {
		// Unix: test with executable permissions
		expectedMode = PermFileExecutable
		testutil.WriteTestFile(t, srcPath, content, expectedMode, "Failed to create source file")
	}

	// Copy the file
	dstPath := filepath.Join(tmpDir, "destination.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination exists
	if FileNotExists(dstPath) {
		t.Fatalf("Destination file does not exist")
	}

	// Verify content matches
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Errorf("Content mismatch: got %q, want %q", string(dstContent), string(content))
	}

	// Verify permissions are preserved (Unix only)
	if !IsWindows() {
		dstInfo, err := os.Stat(dstPath)
		if err != nil {
			t.Fatalf("Failed to stat destination file: %v", err)
		}

		dstMode := dstInfo.Mode().Perm()
		if dstMode != expectedMode {
			t.Errorf("Permissions not preserved: got %o, want %o", dstMode, expectedMode)
		}
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "destination.txt")

	err := CopyFile(srcPath, dstPath)
	if err == nil {
		t.Error("Expected error when source file doesn't exist, got nil")
	}
}

func TestCopyFile_DestinationDirNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")

	// Create source file
	testutil.WriteTestFile(t, srcPath, []byte("test"), PermFileDefault, "Failed to create source file")

	// Try to copy to non-existent directory
	dstPath := filepath.Join(tmpDir, "nonexistent", "destination.txt")
	err := CopyFile(srcPath, dstPath)
	if err == nil {
		t.Error("Expected error when destination directory doesn't exist, got nil")
	}
}

func TestCopyFile_ExecutablePreserved(t *testing.T) {
	if IsWindows() {
		t.Skip("Skipping executable test on Windows")
	}

	tmpDir := t.TempDir()

	// Create an executable file
	srcPath := filepath.Join(tmpDir, "script.sh")
	content := []byte("#!/bin/bash\necho 'test'\n")
	testutil.WriteTestFile(t, srcPath, content, PermFileExecutable, "Failed to create source file")

	// Copy it
	dstPath := filepath.Join(tmpDir, "script-copy.sh")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify it's still executable
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Failed to stat destination: %v", err)
	}

	if dstInfo.Mode().Perm()&0111 == 0 {
		t.Error("Executable bit not preserved")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent file
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	if FileExists(nonExistent) {
		t.Error("FileExists returned true for non-existent file")
	}

	// Create a file
	filePath := filepath.Join(tmpDir, "test.txt")
	testutil.WriteTestFile(t, filePath, []byte("test"), PermFileDefault, "Failed to create test file")

	// Test existing file
	if !FileExists(filePath) {
		t.Error("FileExists returned false for existing file")
	}

	// Test directory (should return false)
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, PermFileExecutable); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	if FileExists(dirPath) {
		t.Error("FileExists returned true for directory")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent directory
	nonExistent := filepath.Join(tmpDir, "nonexistent")
	if DirExists(nonExistent) {
		t.Error("DirExists returned true for non-existent directory")
	}

	// Create a directory
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, PermFileExecutable); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Test existing directory
	if !DirExists(dirPath) {
		t.Error("DirExists returned false for existing directory")
	}

	// Test file (should return false)
	filePath := filepath.Join(tmpDir, "test.txt")
	testutil.WriteTestFile(t, filePath, []byte("test"), PermFileDefault, "Failed to create test file")

	if DirExists(filePath) {
		t.Error("DirExists returned true for file")
	}
}

func TestPathExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent path
	nonExistent := filepath.Join(tmpDir, "nonexistent")
	if PathExists(nonExistent) {
		t.Error("PathExists returned true for non-existent path")
	}

	// Create a file
	filePath := filepath.Join(tmpDir, "test.txt")
	testutil.WriteTestFile(t, filePath, []byte("test"), PermFileDefault, "Failed to create test file")

	// Test existing file
	if !PathExists(filePath) {
		t.Error("PathExists returned false for existing file")
	}

	// Create a directory
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, PermFileExecutable); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Test existing directory
	if !PathExists(dirPath) {
		t.Error("PathExists returned false for existing directory")
	}
}

func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent path
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	info, exists := GetFileInfo(nonExistent)
	if exists {
		t.Error("GetFileInfo returned exists=true for non-existent file")
	}
	if info != nil {
		t.Error("GetFileInfo returned non-nil info for non-existent file")
	}

	// Create a file
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	testutil.WriteTestFile(t, filePath, content, PermFileDefault, "Failed to create test file")

	// Test existing file
	info, exists = GetFileInfo(filePath)
	if !exists {
		t.Error("GetFileInfo returned exists=false for existing file")
	}
	if info == nil {
		t.Fatal("GetFileInfo returned nil info for existing file")
	}

	// Verify info properties
	if info.IsDir() {
		t.Error("FileInfo.IsDir() returned true for file")
	}
	if info.Size() != int64(len(content)) {
		t.Errorf("FileInfo.Size() = %d, want %d", info.Size(), len(content))
	}

	// Test directory
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, PermFileExecutable); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	info, exists = GetFileInfo(dirPath)
	if !exists {
		t.Error("GetFileInfo returned exists=false for existing directory")
	}
	if info == nil {
		t.Fatal("GetFileInfo returned nil info for existing directory")
	}
	if !info.IsDir() {
		t.Error("FileInfo.IsDir() returned false for directory")
	}
}

func TestIsExecutableFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent file
	nonExistent := filepath.Join(tmpDir, "nonexistent")
	if IsExecutableFile(nonExistent) {
		t.Error("IsExecutableFile returned true for non-existent file")
	}

	// Test directory
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, PermFileExecutable); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if IsExecutableFile(dirPath) {
		t.Error("IsExecutableFile returned true for directory")
	}

	// Test regular file (non-executable on Unix)
	regularFile := filepath.Join(tmpDir, "regular.txt")
	testutil.WriteTestFile(t, regularFile, []byte("test"), PermFileDefault, "Failed to create regular file")

	if IsWindows() {
		// On Windows, all files are considered executable if they exist
		if !IsExecutableFile(regularFile) {
			t.Error("IsExecutableFile returned false for regular file on Windows")
		}
	} else {
		// On Unix, non-executable files should return false
		if IsExecutableFile(regularFile) {
			t.Error("IsExecutableFile returned true for non-executable file on Unix")
		}
	}

	// Test executable file (Unix only)
	if !IsWindows() {
		execFile := filepath.Join(tmpDir, "executable.sh")
		testutil.WriteTestFile(t, execFile, []byte("#!/bin/bash\necho test"), PermFileExecutable, "Failed to create executable file")

		if !IsExecutableFile(execFile) {
			t.Error("IsExecutableFile returned false for executable file on Unix")
		}
	}
}

func TestStatWithExistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent path
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	info, exists, err := StatWithExistence(nonExistent)
	if exists {
		t.Error("StatWithExistence returned exists=true for non-existent file")
	}
	if err != nil {
		t.Errorf("StatWithExistence returned error for non-existent file: %v", err)
	}
	if info != nil {
		t.Error("StatWithExistence returned non-nil info for non-existent file")
	}

	// Test existing file
	filePath := filepath.Join(tmpDir, "test.txt")
	testutil.WriteTestFile(t, filePath, []byte("test"), PermFileDefault, "Failed to create test file")

	info, exists, err = StatWithExistence(filePath)
	if !exists {
		t.Error("StatWithExistence returned exists=false for existing file")
	}
	if err != nil {
		t.Errorf("StatWithExistence returned error for existing file: %v", err)
	}
	if info == nil {
		t.Error("StatWithExistence returned nil info for existing file")
	}
}
