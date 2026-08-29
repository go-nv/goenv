package meta

import (
	"fmt"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/version"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest [prefix]",
	Short: "Print the latest installed version matching a prefix",
	Long: `Print the latest version that matches the given prefix.

By default only installed versions are considered. Use --known to search the
list of versions available for download instead.

This command only prints a version — it never changes the active version and
never writes a .go-version file. Use 'goenv use <version>' to switch versions.`,
	Example: `  # Latest installed version
  goenv latest

  # Latest installed 1.21.x
  goenv latest 1.21

  # Latest version available for download
  goenv latest --known

  # Latest 1.21.x available for download
  goenv latest --known 1.21`,
	Args:    cobra.MaximumNArgs(1),
	RunE:    RunLatest,
	GroupID: string(cmdpkg.GroupVersions),
}

// LatestFlags holds flags for the latest command. Exported for testing.
var LatestFlags struct {
	Known bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(latestCmd)
	latestCmd.SilenceUsage = true
	latestCmd.Flags().BoolVarP(&LatestFlags.Known, "known", "k", false,
		"Search versions available for download instead of installed versions")
}

// RunLatest executes the latest command logic. Exported for testing.
func RunLatest(cmd *cobra.Command, args []string) error {
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}

	if LatestFlags.Known {
		return latestKnown(cmd, prefix)
	}
	return latestInstalled(cmd, prefix)
}

func latestInstalled(cmd *cobra.Command, prefix string) error {
	ctx := cmdutil.GetContexts(cmd)
	mgr := ctx.Manager

	spec := prefix
	if spec == "" {
		spec = "latest"
	}

	resolved, err := mgr.ResolveVersionSpec(spec)
	if err != nil {
		if prefix == "" {
			return fmt.Errorf("goenv: no versions installed")
		}
		return fmt.Errorf("goenv: no installed version matches '%s'", prefix)
	}

	fmt.Fprintln(cmd.OutOrStdout(), resolved)
	return nil
}

func latestKnown(cmd *cobra.Command, prefix string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config

	fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: cfg.Debug})
	releases, err := fetcher.FetchWithFallback(cfg.Root)
	if err != nil {
		return errors.FailedTo("get versions", err)
	}

	normalizedPrefix := utils.NormalizeGoVersion(prefix)

	// Releases are returned newest-first, so the first match wins.
	for _, release := range releases {
		if !release.Stable {
			continue
		}
		candidate := strings.TrimPrefix(release.Version, "go")
		if !matchesVersionPrefix(candidate, normalizedPrefix) {
			continue
		}
		fmt.Fprintln(cmd.OutOrStdout(), candidate)
		return nil
	}

	if prefix == "" {
		return fmt.Errorf("goenv: no stable version found")
	}
	return fmt.Errorf("goenv: no known version matches '%s'", prefix)
}

// matchesVersionPrefix reports whether candidate is the version prefix itself
// or a more specific version under it. An empty prefix matches everything.
// This is component-aware so "1.2" does not match "1.21.0".
func matchesVersionPrefix(candidate, prefix string) bool {
	if prefix == "" {
		return true
	}
	candidate = utils.NormalizeGoVersion(candidate)
	return candidate == prefix || strings.HasPrefix(candidate, prefix+".")
}
