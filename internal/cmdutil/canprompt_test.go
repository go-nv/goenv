package cmdutil

import (
	"os"
	"testing"
)

// CanPrompt guards prompts that default to "yes" and whose effect is to write
// to a file the user may have committed — specifically the "Detected VS Code
// workspace. Update settings for Go X? [Y/n]" prompt in 'goenv use' and
// 'goenv local'.
//
// Before this guard existed, those call sites read stdin with fmt.Fscanln and
// treated an empty response as consent. On a non-terminal stdin that read
// returns immediately, so running 'goenv use' in CI silently rewrote a tracked
// .vscode/settings.json. Both halves of the condition are load-bearing, so both
// are pinned here.
func TestCanPrompt(t *testing.T) {
	// A regular file is never a terminal, which is what stdin looks like under
	// CI, cron, container builds, and `goenv use < /dev/null`.
	notATTY, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer notATTY.Close()

	// os.DevNull is the case that matters most and the one that used to slip
	// through: it is a character device, so the old os.ModeCharDevice test
	// classified it as a terminal. Redirecting from /dev/null is precisely how
	// cron jobs and container builds invoke a CLI, and no CI variable is
	// necessarily set in either.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("failed to open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	tests := []struct {
		name  string
		level InteractionLevel
		stdin *os.File
		want  bool
	}{
		{
			name:  "non-interactive level, non-tty stdin",
			level: InteractionNone,
			stdin: notATTY,
			want:  false,
		},
		{
			// The case CI variables miss: no CI env set, so the context looks
			// interactive, but stdin is a pipe or /dev/null. Without the IsTTY
			// half this returns true and the caller prompts into the void.
			name:  "interactive level but stdin is not a terminal",
			level: InteractionMinimal,
			stdin: notATTY,
			want:  false,
		},
		{
			name:  "guided level but stdin is not a terminal",
			level: InteractionGuided,
			stdin: notATTY,
			want:  false,
		},
		{
			name:  "interactive level but stdin redirected from /dev/null",
			level: InteractionMinimal,
			stdin: devNull,
			want:  false,
		},
		{
			name:  "nil stdin",
			level: InteractionMinimal,
			stdin: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &InteractiveContext{Level: tt.level}
			if got := CanPrompt(ctx, tt.stdin); got != tt.want {
				t.Errorf("CanPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// A closed or otherwise unstattable file must not be reported as promptable;
// IsTTY returns false when Stat fails, and CanPrompt must inherit that.
func TestCanPrompt_UnstattableStdin(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close() // Stat on a closed file fails

	ctx := &InteractiveContext{Level: InteractionMinimal}
	if CanPrompt(ctx, f) {
		t.Error("CanPrompt() = true for a closed stdin; must fail closed")
	}
}
