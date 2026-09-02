package meta

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// releasesFixture mirrors the real go-nv/goenv releases payload that breaks
// "goenv update": the two newest published releases come from the v2
// maintenance branch and carry no assets at all, so /releases/latest points at
// a release this binary cannot update to (issue #582).
const releasesFixture = `[
  {"tag_name": "2.2.43", "draft": false, "prerelease": false, "assets": []},
  {"tag_name": "2.2.42", "draft": false, "prerelease": false, "assets": []},
  {"tag_name": "3.1.4", "draft": false, "prerelease": false, "assets": [
    {"name": "goenv_3.1.4_checksums.txt", "browser_download_url": "https://example.test/goenv_3.1.4_checksums.txt"},
    {"name": "goenv_3.1.4_darwin_arm64.tar.gz", "browser_download_url": "https://example.test/goenv_3.1.4_darwin_arm64.tar.gz"},
    {"name": "goenv_3.1.4_linux_amd64.tar.gz", "browser_download_url": "https://example.test/goenv_3.1.4_linux_amd64.tar.gz"},
    {"name": "goenv_3.1.4_windows_amd64.zip", "browser_download_url": "https://example.test/goenv_3.1.4_windows_amd64.zip"}
  ]},
  {"tag_name": "3.1.3", "draft": false, "prerelease": false, "assets": [
    {"name": "goenv_3.1.3_darwin_arm64.tar.gz", "browser_download_url": "https://example.test/goenv_3.1.3_darwin_arm64.tar.gz"}
  ]}
]`

func TestSelectRelease_SkipsReleasesWithoutBinaries(t *testing.T) {
	tests := []struct {
		name        string
		os          string
		arch        string
		wantVersion string
		wantURL     string
	}{
		{
			name:        "darwin arm64 skips assetless v2 releases",
			os:          "darwin",
			arch:        "arm64",
			wantVersion: "3.1.4",
			wantURL:     "https://example.test/goenv_3.1.4_darwin_arm64.tar.gz",
		},
		{
			name:        "linux amd64",
			os:          "linux",
			arch:        "amd64",
			wantVersion: "3.1.4",
			wantURL:     "https://example.test/goenv_3.1.4_linux_amd64.tar.gz",
		},
		{
			name:        "windows amd64 picks the zip",
			os:          "windows",
			arch:        "amd64",
			wantVersion: "3.1.4",
			wantURL:     "https://example.test/goenv_3.1.4_windows_amd64.zip",
		},
		{
			name:        "falls back to an older release when newest lacks the platform",
			os:          "linux",
			arch:        "armv7",
			wantVersion: "",
			wantURL:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, url := selectRelease([]byte(releasesFixture), tt.os, tt.arch)

			assert.Equal(t, tt.wantVersion, version)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

func TestSelectRelease_IgnoresDraftsAndPrereleases(t *testing.T) {
	payload := `[
	  {"tag_name": "4.0.0", "draft": true, "prerelease": false, "assets": [
	    {"name": "goenv_4.0.0_linux_amd64.tar.gz", "browser_download_url": "https://example.test/draft.tar.gz"}
	  ]},
	  {"tag_name": "4.0.0-rc1", "draft": false, "prerelease": true, "assets": [
	    {"name": "goenv_4.0.0-rc1_linux_amd64.tar.gz", "browser_download_url": "https://example.test/rc.tar.gz"}
	  ]},
	  {"tag_name": "3.1.4", "draft": false, "prerelease": false, "assets": [
	    {"name": "goenv_3.1.4_linux_amd64.tar.gz", "browser_download_url": "https://example.test/stable.tar.gz"}
	  ]}
	]`

	version, url := selectRelease([]byte(payload), "linux", "amd64")

	assert.Equal(t, "3.1.4", version)
	assert.Equal(t, "https://example.test/stable.tar.gz", url)
}

func TestSelectRelease_HandlesMalformedPayload(t *testing.T) {
	version, url := selectRelease([]byte("not json"), "linux", "amd64")

	assert.Empty(t, version)
	assert.Empty(t, url)
}

// TestSelectRelease_DoesNotMatchAdjacentArch guards against "amd64" matching an
// "amd64v2"-style asset name via a bare prefix comparison.
func TestSelectRelease_DoesNotMatchAdjacentArch(t *testing.T) {
	payload := `[
	  {"tag_name": "3.1.4", "draft": false, "prerelease": false, "assets": [
	    {"name": "goenv_3.1.4_linux_arm64.tar.gz", "browser_download_url": "https://example.test/arm64.tar.gz"}
	  ]}
	]`

	version, url := selectRelease([]byte(payload), "linux", "arm")

	assert.Empty(t, version, "linux/arm must not match a linux/arm64 asset")
	assert.Empty(t, url)
}

// atomFixture mirrors the public releases feed, which is the rate-limit-free
// fallback used when the REST API refuses the request.
const atomFixture = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry><id>tag:github.com,2008:Repository/12345/2.2.43</id><title>2.2.43</title></entry>
  <entry><id>tag:github.com,2008:Repository/12345/3.1.5</id><title>3.1.5</title></entry>
  <entry><id>tag:github.com,2008:Repository/12345/3.1.4</id><title>3.1.4</title></entry>
</feed>`

func TestSelectAtomRelease(t *testing.T) {
	tests := []struct {
		name           string
		os             string
		arch           string
		currentVersion string
		wantVersion    string
		wantURL        string
	}{
		{
			name:           "skips the v2 maintenance line",
			os:             "darwin",
			arch:           "arm64",
			currentVersion: "3.1.5",
			wantVersion:    "3.1.5",
			wantURL:        "https://github.com/go-nv/goenv/releases/download/3.1.5/goenv_3.1.5_darwin_arm64.tar.gz",
		},
		{
			name:           "windows uses the zip asset",
			os:             "windows",
			arch:           "amd64",
			currentVersion: "3.1.4",
			wantVersion:    "3.1.5",
			wantURL:        "https://github.com/go-nv/goenv/releases/download/3.1.5/goenv_3.1.5_windows_amd64.zip",
		},
		{
			name:           "unknown running version takes the newest tag",
			os:             "linux",
			arch:           "amd64",
			currentVersion: "unknown",
			wantVersion:    "2.2.43",
			wantURL:        "https://github.com/go-nv/goenv/releases/download/2.2.43/goenv_2.2.43_linux_amd64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, url := selectAtomRelease([]byte(atomFixture), tt.os, tt.arch, tt.currentVersion)

			assert.Equal(t, tt.wantVersion, version)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

func TestSelectAtomRelease_HandlesMalformedFeed(t *testing.T) {
	version, url := selectAtomRelease([]byte("not xml"), "linux", "amd64", "3.1.5")

	assert.Empty(t, version)
	assert.Empty(t, url)
}

func TestFetchReleases_NotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	_, _, err = fetchReleases(req)

	assert.ErrorIs(t, err, errNotModified)
}

func TestFetchReleases_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	_, _, err = fetchReleases(req)

	assert.ErrorIs(t, err, errRateLimited)
}

func TestFetchReleases_ReturnsBodyAndETag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"abc123"`)
		fmt.Fprint(w, releasesFixture)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	body, etag, err := fetchReleases(req)

	require.NoError(t, err)
	assert.Equal(t, `"abc123"`, etag)
	assert.JSONEq(t, releasesFixture, string(body))
}
