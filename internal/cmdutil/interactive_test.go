package cmdutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
)

func TestNewInteractiveContext(t *testing.T) {
	// Clear CI environment to ensure consistent test behavior
	defer testutil.ClearCIEnvironment(t)()

	tests := []struct {
		name          string
		setupCmd      func() *cobra.Command
		expectedLevel InteractionLevel
		expectedYes   bool
		expectedQuiet bool
	}{
		{
			name: "default - minimal interaction",
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("interactive", false, "")
				cmd.Flags().Bool("yes", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("force", false, "")
				return cmd
			},
			expectedLevel: InteractionMinimal,
			expectedYes:   false,
			expectedQuiet: false,
		},
		{
			name: "with --interactive flag",
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("interactive", false, "")
				cmd.Flags().Bool("yes", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Set("interactive", "true")
				return cmd
			},
			expectedLevel: InteractionGuided,
			expectedYes:   false,
			expectedQuiet: false,
		},
		{
			name: "with --yes flag",
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("interactive", false, "")
				cmd.Flags().Bool("yes", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Set("yes", "true")
				return cmd
			},
			expectedLevel: InteractionNone,
			expectedYes:   true,
			expectedQuiet: false,
		},
		{
			name: "with --force flag",
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("interactive", false, "")
				cmd.Flags().Bool("yes", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Set("force", "true")
				return cmd
			},
			expectedLevel: InteractionNone,
			expectedYes:   true,
			expectedQuiet: false,
		},
		{
			name: "with --quiet flag",
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("interactive", false, "")
				cmd.Flags().Bool("yes", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Set("quiet", "true")
				return cmd
			},
			expectedLevel: InteractionNone,
			expectedYes:   false,
			expectedQuiet: true,
		},
		{
			name: "with --yes and --quiet",
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("interactive", false, "")
				cmd.Flags().Bool("yes", false, "")
				cmd.Flags().Bool("quiet", false, "")
				cmd.Flags().Bool("force", false, "")
				cmd.Flags().Set("yes", "true")
				cmd.Flags().Set("quiet", "true")
				return cmd
			},
			expectedLevel: InteractionNone,
			expectedYes:   true,
			expectedQuiet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setupCmd()
			ctx := NewInteractiveContext(cmd)

			if ctx.Level != tt.expectedLevel {
				t.Errorf("Level = %v, want %v", ctx.Level, tt.expectedLevel)
			}
			if ctx.AssumeYes != tt.expectedYes {
				t.Errorf("AssumeYes = %v, want %v", ctx.AssumeYes, tt.expectedYes)
			}
			if ctx.Quiet != tt.expectedQuiet {
				t.Errorf("Quiet = %v, want %v", ctx.Quiet, tt.expectedQuiet)
			}
		})
	}
}

func TestInteractiveContext_IsInteractive(t *testing.T) {
	tests := []struct {
		name  string
		level InteractionLevel
		want  bool
	}{
		{"none", InteractionNone, false},
		{"minimal", InteractionMinimal, true},
		{"guided", InteractionGuided, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &InteractiveContext{Level: tt.level}
			if got := ctx.IsInteractive(); got != tt.want {
				t.Errorf("IsInteractive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveContext_IsGuided(t *testing.T) {
	tests := []struct {
		name  string
		level InteractionLevel
		want  bool
	}{
		{"none", InteractionNone, false},
		{"minimal", InteractionMinimal, false},
		{"guided", InteractionGuided, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &InteractiveContext{Level: tt.level}
			if got := ctx.IsGuided(); got != tt.want {
				t.Errorf("IsGuided() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveContext_Confirm(t *testing.T) {
	tests := []struct {
		name       string
		level      InteractionLevel
		assumeYes  bool
		input      string
		defaultYes bool
		want       bool
	}{
		{
			name:       "assume yes always confirms",
			level:      InteractionMinimal,
			assumeYes:  true,
			input:      "",
			defaultYes: false,
			want:       true,
		},
		{
			name:       "non-interactive returns default (true)",
			level:      InteractionNone,
			assumeYes:  false,
			input:      "",
			defaultYes: true,
			want:       true,
		},
		{
			name:       "non-interactive returns default (false)",
			level:      InteractionNone,
			assumeYes:  false,
			input:      "",
			defaultYes: false,
			want:       false,
		},
		{
			name:       "user says yes",
			level:      InteractionMinimal,
			assumeYes:  false,
			input:      "y\n",
			defaultYes: false,
			want:       true,
		},
		{
			name:       "user says no",
			level:      InteractionMinimal,
			assumeYes:  false,
			input:      "n\n",
			defaultYes: true,
			want:       false,
		},
		{
			name:       "empty input with default yes",
			level:      InteractionMinimal,
			assumeYes:  false,
			input:      "\n",
			defaultYes: true,
			want:       true,
		},
		{
			name:       "empty input with default no",
			level:      InteractionMinimal,
			assumeYes:  false,
			input:      "\n",
			defaultYes: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			ctx := &InteractiveContext{
				Level:     tt.level,
				AssumeYes: tt.assumeYes,
				Reader:    strings.NewReader(tt.input),
				Writer:    &stdout,
				ErrWriter: &stderr,
			}

			got := ctx.Confirm("Test question?", tt.defaultYes)
			if got != tt.want {
				t.Errorf("Confirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveContext_OfferRepair(t *testing.T) {
	tests := []struct {
		name              string
		level             InteractionLevel
		assumeYes         bool
		quiet             bool
		input             string
		want              bool
		wantOutputContain string
		wantErrContain    string
	}{
		{
			name:              "assume yes auto-repairs",
			level:             InteractionMinimal,
			assumeYes:         true,
			quiet:             false,
			input:             "",
			want:              true,
			wantOutputContain: "Attempting repair",
		},
		{
			name:              "assume yes quiet mode",
			level:             InteractionMinimal,
			assumeYes:         true,
			quiet:             true,
			input:             "",
			want:              true,
			wantOutputContain: "",
		},
		{
			name:           "non-interactive shows error",
			level:          InteractionNone,
			assumeYes:      false,
			quiet:          false,
			input:          "",
			want:           false,
			wantErrContain: "non-interactive mode",
		},
		{
			name:              "user accepts repair",
			level:             InteractionMinimal,
			assumeYes:         false,
			quiet:             false,
			input:             "y\n",
			want:              true,
			wantOutputContain: "Attempting repair",
		},
		{
			name:      "user declines repair",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "n\n",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			ctx := &InteractiveContext{
				Level:     tt.level,
				AssumeYes: tt.assumeYes,
				Quiet:     tt.quiet,
				Reader:    strings.NewReader(tt.input),
				Writer:    &stdout,
				ErrWriter: &stderr,
			}

			got := ctx.OfferRepair("Test problem", "Test repair")
			if got != tt.want {
				t.Errorf("OfferRepair() = %v, want %v", got, tt.want)
			}

			if tt.wantOutputContain != "" && !strings.Contains(stdout.String(), tt.wantOutputContain) {
				t.Errorf("Output should contain %q, got: %s", tt.wantOutputContain, stdout.String())
			}

			if tt.wantErrContain != "" && !strings.Contains(stderr.String(), tt.wantErrContain) {
				t.Errorf("Error output should contain %q, got: %s", tt.wantErrContain, stderr.String())
			}
		})
	}
}

func TestInteractiveContext_Select(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}

	tests := []struct {
		name      string
		level     InteractionLevel
		assumeYes bool
		quiet     bool
		input     string
		want      int
	}{
		{
			name:      "assume yes selects first",
			level:     InteractionMinimal,
			assumeYes: true,
			quiet:     false,
			input:     "",
			want:      1,
		},
		{
			name:      "non-interactive returns 0",
			level:     InteractionNone,
			assumeYes: false,
			quiet:     false,
			input:     "",
			want:      0,
		},
		{
			name:      "user selects option 2",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "2\n",
			want:      2,
		},
		{
			name:      "user selects option 3",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "3\n",
			want:      3,
		},
		{
			name:      "invalid selection (too high)",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "5\n",
			want:      0,
		},
		{
			name:      "invalid selection (zero)",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "0\n",
			want:      0,
		},
		{
			name:      "invalid selection (negative)",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "-1\n",
			want:      0,
		},
		{
			name:      "invalid selection (not a number)",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "abc\n",
			want:      0,
		},
		{
			name:      "empty input",
			level:     InteractionMinimal,
			assumeYes: false,
			quiet:     false,
			input:     "\n",
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			ctx := &InteractiveContext{
				Level:     tt.level,
				AssumeYes: tt.assumeYes,
				Quiet:     tt.quiet,
				Reader:    strings.NewReader(tt.input),
				Writer:    &stdout,
				ErrWriter: &stderr,
			}

			got := ctx.Select("Test question?", options)
			if got != tt.want {
				t.Errorf("Select() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveContext_Printf(t *testing.T) {
	tests := []struct {
		name      string
		quiet     bool
		wantEmpty bool
	}{
		{"normal mode prints", false, false},
		{"quiet mode suppresses", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			ctx := &InteractiveContext{
				Quiet:  tt.quiet,
				Writer: &stdout,
			}

			ctx.Printf("test output %s", "here")

			if tt.wantEmpty && stdout.Len() > 0 {
				t.Errorf("Expected empty output in quiet mode, got: %s", stdout.String())
			}
			if !tt.wantEmpty && stdout.Len() == 0 {
				t.Error("Expected output but got none")
			}
		})
	}
}

func TestInteractiveContext_ErrorPrintf(t *testing.T) {
	tests := []struct {
		name  string
		quiet bool
	}{
		{"normal mode prints", false},
		{"quiet mode still prints errors", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			ctx := &InteractiveContext{
				Quiet:     tt.quiet,
				ErrWriter: &stderr,
			}

			ctx.ErrorPrintf("error output")

			if stderr.Len() == 0 {
				t.Error("Expected error output but got none")
			}
		})
	}
}

func TestIsCI(t *testing.T) {
	// Save original env vars
	ciVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"BUILD_ID",
		"BUILDKITE",
		"DRONE",
		"TEAMCITY_VERSION",
	}
	// Clear all CI vars at the start
	for _, v := range ciVars {
		t.Setenv(v, "")
	}

	// Test with no CI vars
	if IsCI() {
		t.Error("IsCI() = true with no CI vars, want false")
	}

	// Test each CI var individually
	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{"GitHub Actions", "GITHUB_ACTIONS", "true"},
		{"GitLab CI", "GITLAB_CI", "true"},
		{"CircleCI", "CIRCLECI", "true"},
		{"Travis CI", "TRAVIS", "true"},
		{"Generic CI", "CI", "true"},
		{"Jenkins", "BUILD_ID", "123"},
		{"Buildkite", "BUILDKITE", "true"},
		{"Drone", "DRONE", "true"},
		{"TeamCity", "TEAMCITY_VERSION", "2023.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all CI vars
			for _, v := range ciVars {
				t.Setenv(v, "")
			}

			// Set the specific CI var
			t.Setenv(tt.envVar, tt.value)

			if !IsCI() {
				t.Errorf("IsCI() = false with %s=%s, want true", tt.envVar, tt.value)
			}
		})
	}
}
