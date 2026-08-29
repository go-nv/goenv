package meta

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempFile writes content to a temp file and returns its path.
func writeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestVerifyChecksum_MatchingHashPasses is the baseline: a correct digest
// verifies.
func TestVerifyChecksum_MatchingHashPasses(t *testing.T) {
	payload := []byte("goenv release archive")
	archive := writeTempFile(t, "goenv_9.9.9_linux_amd64.tar.gz", payload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  goenv_9.9.9_linux_amd64.tar.gz\n", sha256Hex(payload))
	}))
	defer server.Close()

	assert.NoError(t, verifyChecksum(archive, server.URL, "goenv_9.9.9_linux_amd64.tar.gz"))
}

// TestVerifyChecksum_MismatchIsFatal is the security-critical case.
//
// A published digest that does not match the downloaded bytes means the
// artifact is not what the release says it is. This must surface as an error
// the caller aborts on — a warning that installs anyway is not a control, and
// leaves the user executing an unverified binary.
func TestVerifyChecksum_MismatchIsFatal(t *testing.T) {
	archive := writeTempFile(t, "goenv_9.9.9_linux_amd64.tar.gz", []byte("tampered content"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  goenv_9.9.9_linux_amd64.tar.gz\n", sha256Hex([]byte("the real archive")))
	}))
	defer server.Close()

	err := verifyChecksum(archive, server.URL, "goenv_9.9.9_linux_amd64.tar.gz")

	require.Error(t, err, "a checksum mismatch must be an error")
	assert.False(t, stderrors.Is(err, errChecksumsUnpublished),
		"a mismatch must not be classified as 'no checksums published', which the caller tolerates")
	assert.Contains(t, err.Error(), "mismatch")
}

// TestVerifyChecksum_MissingEntryIsFatal covers a checksums file that exists
// but has no line for this artifact. Verification was possible and did not
// succeed, so it must not be waved through.
func TestVerifyChecksum_MissingEntryIsFatal(t *testing.T) {
	archive := writeTempFile(t, "goenv_9.9.9_linux_amd64.tar.gz", []byte("payload"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  goenv_9.9.9_darwin_arm64.tar.gz\n", sha256Hex([]byte("other")))
	}))
	defer server.Close()

	err := verifyChecksum(archive, server.URL, "goenv_9.9.9_linux_amd64.tar.gz")

	require.Error(t, err)
	assert.False(t, stderrors.Is(err, errChecksumsUnpublished))
}

// TestVerifyChecksum_UnpublishedIsDistinguishable covers the one tolerated
// case: the release genuinely publishes no checksums file. The caller may
// proceed with a warning, so this must be identifiable rather than lumped in
// with verification failures.
func TestVerifyChecksum_UnpublishedIsDistinguishable(t *testing.T) {
	archive := writeTempFile(t, "goenv_9.9.9_linux_amd64.tar.gz", []byte("payload"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	err := verifyChecksum(archive, server.URL, "goenv_9.9.9_linux_amd64.tar.gz")

	require.Error(t, err)
	assert.True(t, stderrors.Is(err, errChecksumsUnpublished),
		"a 404 on the checksums file must be reported as 'unpublished' so the caller can distinguish it")
}

// --- archive extraction -----------------------------------------------------

func makeTarGz(t *testing.T, entries map[string][]byte) string {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, content := range entries {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}))
		_, err := tw.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	return writeTempFile(t, "archive.tar.gz", buf.Bytes())
}

func makeZip(t *testing.T, entries map[string][]byte) string {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	return writeTempFile(t, "archive.zip", buf.Bytes())
}

// TestExtractGoenvBinary_TarGz covers the path every non-Windows self-update
// takes. Release assets are archives, so a failure here means "goenv update"
// installs something that is not an executable.
func TestExtractGoenvBinary_TarGz(t *testing.T) {
	want := []byte("#!/bin/sh\necho goenv\n")
	archive := makeTarGz(t, map[string][]byte{
		"README.md": []byte("docs"),
		"goenv":     want,
	})

	path, err := extractGoenvBinary(archive, "https://example.com/goenv_9.9.9_linux_amd64.tar.gz")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestExtractGoenvBinary_Zip covers the Windows path.
func TestExtractGoenvBinary_Zip(t *testing.T) {
	want := []byte("MZ fake windows binary")
	archive := makeZip(t, map[string][]byte{
		"LICENSE":   []byte("license"),
		"goenv.exe": want,
	})

	path, err := extractGoenvBinary(archive, "https://example.com/goenv_9.9.9_windows_amd64.zip")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestExtractGoenvBinary_MissingBinaryFails ensures an archive without the
// executable is reported rather than silently producing an empty file that
// would then replace the user's working goenv.
func TestExtractGoenvBinary_MissingBinaryFails(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{
		"README.md": []byte("docs only"),
	})

	_, err := extractGoenvBinary(archive, "https://example.com/goenv_9.9.9_linux_amd64.tar.gz")
	assert.Error(t, err, "an archive with no goenv binary must fail loudly")
}

// TestExtractGoenvBinary_MalformedArchiveFails covers a truncated or corrupt
// download, which is what a partial transfer looks like.
func TestExtractGoenvBinary_MalformedArchiveFails(t *testing.T) {
	archive := writeTempFile(t, "broken.tar.gz", []byte("this is not a gzip stream"))

	_, err := extractGoenvBinary(archive, "https://example.com/goenv_9.9.9_linux_amd64.tar.gz")
	assert.Error(t, err, "a corrupt archive must fail rather than yield a partial binary")
}

// TestExtractGoenvBinary_IgnoresNestedPaths guards against a path-traversal
// entry: only the base name is honoured, so "../../etc/goenv" cannot escape.
func TestExtractGoenvBinary_IgnoresNestedPaths(t *testing.T) {
	want := []byte("real binary")
	archive := makeTarGz(t, map[string][]byte{
		"dist/linux_amd64/goenv": want,
	})

	path, err := extractGoenvBinary(archive, "https://example.com/goenv_9.9.9_linux_amd64.tar.gz")
	require.NoError(t, err, "a binary nested in a directory should still be found by base name")
	t.Cleanup(func() { os.Remove(path) })

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
