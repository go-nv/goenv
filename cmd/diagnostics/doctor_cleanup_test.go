package diagnostics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCleanupRecommendation_AgreesWithTheKeepMarker is a regression test for
// contradictory advice in the duplicate-installation cleanup.
//
// Each installation is listed with either "[RECOMMENDED TO KEEP]" or
// "[can safely remove]", and a summary line advises what to do. Those were
// computed independently and disagreed: with a manual install first in PATH and
// a Homebrew install second, the list marked the manual one to keep while the
// summary said to keep "only Homebrew".
//
// This matters because the advice is printed immediately above a prompt that
// deletes files. Following it meant removing the goenv actually being executed
// and keeping an older one. Advice attached to a destructive action must name
// the same thing the action will spare.
func TestCleanupRecommendation_AgreesWithTheKeepMarker(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
	}{
		{
			name: "manual first, homebrew second",
			paths: []string{
				"/Users/someone/.goenv/bin/goenv",
				"/opt/homebrew/Cellar/goenv/3.1.4/bin/goenv",
			},
		},
		{
			name: "homebrew first, manual second",
			paths: []string{
				"/opt/homebrew/Cellar/goenv/3.1.4/bin/goenv",
				"/Users/someone/.goenv/bin/goenv",
			},
		},
		{
			name: "manual plus system",
			paths: []string{
				"/Users/someone/.goenv/bin/goenv",
				"/usr/local/bin/goenv",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := classifyInstallations(tt.paths)
			require.Len(t, classified, len(tt.paths))

			// Exactly one installation must be marked to keep, otherwise the
			// prompt cannot present a coherent choice.
			var kept []string
			for _, inst := range classified {
				if inst.recommended {
					kept = append(kept, inst.path)
				}
			}
			require.Len(t, kept, 1, "exactly one installation should be recommended to keep")

			recommendation := generateCleanupRecommendation(classified)
			require.NotEmpty(t, recommendation)

			assert.Contains(t, recommendation, kept[0],
				"the summary advice must name the same installation the list marks "+
					"[RECOMMENDED TO KEEP]; disagreeing advice above a deletion prompt "+
					"leads users to remove the binary they are running")

			// And it must not advise removing the very thing it says to keep.
			for _, inst := range classified {
				if inst.recommended {
					continue
				}
				if strings.Contains(recommendation, "Keep "+inst.path) {
					t.Fatalf("advice says to keep %q, which is marked removable", inst.path)
				}
			}
		})
	}
}

// TestCleanupRecommendation_SingleInstallationIsSilent keeps the check quiet
// when there is nothing to choose between.
func TestCleanupRecommendation_SingleInstallationIsSilent(t *testing.T) {
	classified := classifyInstallations([]string{"/Users/someone/.goenv/bin/goenv"})
	assert.Empty(t, generateCleanupRecommendation(classified))
}
