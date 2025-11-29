package install

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/schollz/progressbar/v3"
)

// RetryConfig holds configuration for download retries
type RetryConfig struct {
	MaxRetries       int           // Maximum number of retry attempts
	InitialDelay     time.Duration // Initial delay before first retry
	MaxDelay         time.Duration // Maximum delay between retries
	Timeout          time.Duration // Overall timeout for download
	EnableResume     bool          // Whether to use HTTP Range for resume
	InteractiveRetry bool          // Whether to prompt user for retry
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	// Check environment variables for customization
	maxRetries := 3
	if env := utils.GoenvEnvVarInstallRetries.UnsafeValue(); env != "" {
		fmt.Sscanf(env, "%d", &maxRetries)
		if maxRetries < 1 {
			maxRetries = 3
		}
	}

	timeout := 10 * time.Minute
	if env := utils.GoenvEnvVarInstallTimeout.UnsafeValue(); env != "" {
		var seconds int
		fmt.Sscanf(env, "%d", &seconds)
		if seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	enableResume := true
	if env := utils.GoenvEnvVarInstallResume.UnsafeValue(); env == "0" || env == "false" {
		enableResume = false
	}

	return RetryConfig{
		MaxRetries:       maxRetries,
		InitialDelay:     time.Second,
		MaxDelay:         30 * time.Second,
		Timeout:          timeout,
		EnableResume:     enableResume,
		InteractiveRetry: false, // Set by command flag
	}
}

// DownloadWithRetry downloads a file with retry logic and optional resume
func (i *Installer) DownloadWithRetry(url, expectedSHA256, filename string, retryConfig RetryConfig) (string, error) {
	var lastErr error
	var attempt int

	for attempt = 1; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 1 {
			delay := calculateBackoff(attempt, retryConfig.InitialDelay, retryConfig.MaxDelay)

			if !i.Quiet {
				fmt.Fprintf(os.Stderr, "\n%s Retry attempt %d/%d in %v...\n",
					utils.Emoji("🔄"), attempt, retryConfig.MaxRetries, delay)
			}

			time.Sleep(delay)
		}

		// Attempt download
		tempFile, err := i.downloadFileWithResume(url, expectedSHA256, filename, retryConfig, attempt)
		if err == nil {
			if attempt > 1 && !i.Quiet {
				fmt.Fprintf(os.Stderr, "%s Download succeeded on retry attempt %d\n", utils.Emoji("✅"), attempt)
			}
			return tempFile, nil
		}

		lastErr = err

		// Check if this is a retryable error
		if !isRetryableError(err) {
			return "", fmt.Errorf("non-retryable error: %w", err)
		}

		if !i.Quiet {
			fmt.Fprintf(os.Stderr, "%s Download failed: %v\n", utils.Emoji("❌"), err)
		}
	}

	// All retries exhausted
	return "", fmt.Errorf("download failed after %d attempts: %w", attempt-1, lastErr)
}

// downloadFileWithResume attempts to download with optional HTTP Range support
func (i *Installer) downloadFileWithResume(url, expectedSHA256, filename string, retryConfig RetryConfig, attempt int) (string, error) {
	// Create/open temporary file
	tempFilePath := fmt.Sprintf("%s/goenv-download-%s.tmp", os.TempDir(), filename)

	var existingSize int64
	var tempFile *os.File
	var err error

	// Check if partial download exists and resume is enabled
	if retryConfig.EnableResume && attempt > 1 {
		existingSize = utils.GetFileSize(tempFilePath)
		if existingSize > 0 {
			tempFile, err = os.OpenFile(tempFilePath, os.O_APPEND|os.O_WRONLY, utils.PermFileDefault)
			if err != nil {
				// Can't resume, start fresh
				existingSize = 0
				tempFile, err = os.Create(tempFilePath)
				if err != nil {
					return "", errors.FailedTo("create temp file", err)
				}
			} else if !i.Quiet {
				fmt.Fprintf(os.Stderr, "%s Resuming download from byte %d...\n", utils.Emoji("▶️ "), existingSize)
			}
		}
	}

	// Create new file if not resuming
	if tempFile == nil {
		tempFile, err = os.Create(tempFilePath)
		if err != nil {
			return "", errors.FailedTo("create temp file", err)
		}
	}
	defer tempFile.Close()

	// Create HTTP request with timeout
	client := utils.NewHTTPClient(retryConfig.Timeout)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.FailedTo("create request", err)
	}

	// Add Range header for resume if applicable
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		// If resume failed, try fresh download
		if existingSize > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			tempFile.Close()
			os.Remove(tempFilePath)
			return i.downloadFileWithResume(url, expectedSHA256, filename, RetryConfig{
				MaxRetries:   1,
				EnableResume: false,
				Timeout:      retryConfig.Timeout,
			}, 1)
		}
		return "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	// Create hash writer to verify checksum
	hasher := sha256.New()

	// If resuming, we need to hash the existing content first
	if existingSize > 0 {
		tempFile.Seek(0, 0)
		if _, err := io.Copy(hasher, tempFile); err != nil {
			return "", errors.FailedTo("hash existing data", err)
		}
		tempFile.Seek(0, 2) // Seek to end for appending
	}

	// Setup progress bar
	var writer io.Writer
	var bar *progressbar.ProgressBar

	totalSize := resp.ContentLength
	if resp.StatusCode == http.StatusPartialContent {
		totalSize += existingSize
	}

	if !i.Quiet && totalSize > 0 {
		bar = progressbar.NewOptions64(
			totalSize,
			progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", filename)),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stderr, "\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetRenderBlankState(true),
		)

		// Set existing progress if resuming
		if existingSize > 0 {
			bar.Set64(existingSize)
		}

		writer = io.MultiWriter(tempFile, hasher, bar)
	} else {
		writer = io.MultiWriter(tempFile, hasher)
		if !i.Quiet {
			if existingSize > 0 {
				fmt.Fprintf(os.Stderr, "Resuming download of %s from byte %d...\n", filename, existingSize)
			} else {
				fmt.Fprintf(os.Stderr, "Downloading %s...\n", filename)
			}
		}
	}

	// Download
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return "", errors.FailedTo("write download", err)
	}

	// Verify checksum
	actualSHA256 := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		os.Remove(tempFilePath)
		return "", &InstallError{
			Type:    ErrorTypeChecksum,
			Phase:   PhaseVerifying,
			Message: "checksum verification failed",
			Err:     fmt.Errorf("expected %s, got %s", expectedSHA256, actualSHA256),
		}
	}

	if !i.Quiet && i.Verbose {
		fmt.Fprintln(os.Stderr, "Download completed and verified")
	}

	return tempFilePath, nil
}

// calculateBackoff calculates exponential backoff
func calculateBackoff(attempt int, initialDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: initialDelay * 2^(attempt-1)
	delay := initialDelay * time.Duration(1<<uint(attempt-2))

	// Cap at maxDelay
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// isRetryableError determines if an error is worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors that are typically transient
	retryablePatterns := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"temporary failure",
		"no such host", // DNS lookup failures
		"TLS handshake timeout",
		"EOF",
		"broken pipe",
		"i/o timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
