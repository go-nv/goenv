package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// SHA256String computes the SHA256 hash of a string and returns it as a hex string.
// This consolidates the repeated pattern of sha256.Sum256 + hex encoding (10+ occurrences).
//
// Example:
//
//	hash := SHA256String("hello world")
//	// Returns: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
func SHA256String(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// SHA256Bytes computes the SHA256 hash of bytes and returns it as a hex string.
func SHA256Bytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA256File computes the SHA256 hash of a file's contents.
// This consolidates the repeated pattern of opening file + hashing (6+ occurrences).
//
// Example:
//
//	hash, err := SHA256File("/path/to/file")
//	if err != nil {
//	    return err
//	}
//	fmt.Println("SHA256:", hash)
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// SHA256Reader computes the SHA256 hash of data from a reader.
// Useful for hashing data that's being streamed or already in a reader.
func SHA256Reader(r io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("failed to hash reader: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyFileSHA256 verifies that a file's SHA256 hash matches the expected hash.
// Returns nil if the hash matches, or an error if it doesn't match or can't be computed.
//
// Example:
//
//	err := VerifyFileSHA256("/path/to/file", "expected_hash_here")
//	if err != nil {
//	    fmt.Println("Verification failed:", err)
//	}
func VerifyFileSHA256(path, expectedHash string) error {
	actualHash, err := SHA256File(path)
	if err != nil {
		return err
	}

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}
