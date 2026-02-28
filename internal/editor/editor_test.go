package editor_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bilustek/gosecrets/internal/editor"
)

// ---------------------------------------------------------------------------
// Task 2: New constructor and options
// ---------------------------------------------------------------------------

func TestNewUsesDefaultCmd(t *testing.T) {
	t.Parallel()

	e, err := editor.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if e.Cmd() == "" {
		t.Fatal("expected Cmd() to be non-empty when using defaults")
	}
}

func TestNewResolvesEditorEnv(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	t.Setenv("VISUAL", "")

	e, err := editor.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := e.Cmd(); got != "nano" {
		t.Fatalf("expected Cmd() == %q, got %q", "nano", got)
	}
}

func TestNewResolvesVisualEnv(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "code --wait")

	e, err := editor.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := e.Cmd(); got != "code --wait" {
		t.Fatalf("expected Cmd() == %q, got %q", "code --wait", got)
	}
}

func TestNewFallsBackToVi(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	e, err := editor.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := e.Cmd(); got != "vi" {
		t.Fatalf("expected Cmd() == %q, got %q", "vi", got)
	}
}

func TestNewEditorPrecedence(t *testing.T) {
	t.Setenv("EDITOR", "emacs")
	t.Setenv("VISUAL", "code")

	e, err := editor.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := e.Cmd(); got != "emacs" {
		t.Fatalf("expected EDITOR to take precedence, got %q", got)
	}
}

func TestNewWithCmd(t *testing.T) {
	t.Parallel()

	e, err := editor.New(editor.WithCmd("nvim"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := e.Cmd(); got != "nvim" {
		t.Fatalf("expected Cmd() == %q, got %q", "nvim", got)
	}
}

func TestNewRejectsEmptyCmd(t *testing.T) {
	t.Parallel()

	if _, err := editor.New(editor.WithCmd("")); err == nil {
		t.Fatal("expected error for empty cmd, got nil")
	} else if got := err.Error(); !contains(got, "cmd cannot be empty") {
		t.Fatalf("expected error to contain %q, got %q", "cmd cannot be empty", got)
	}
}

func TestNewRejectsNilStdin(t *testing.T) {
	t.Parallel()

	if _, err := editor.New(editor.WithStdin(nil)); err == nil {
		t.Fatal("expected error for nil stdin, got nil")
	} else if got := err.Error(); !contains(got, "stdin cannot be nil") {
		t.Fatalf("expected error to contain %q, got %q", "stdin cannot be nil", got)
	}
}

func TestNewRejectsNilStdout(t *testing.T) {
	t.Parallel()

	if _, err := editor.New(editor.WithStdout(nil)); err == nil {
		t.Fatal("expected error for nil stdout, got nil")
	} else if got := err.Error(); !contains(got, "stdout cannot be nil") {
		t.Fatalf("expected error to contain %q, got %q", "stdout cannot be nil", got)
	}
}

func TestNewRejectsNilStderr(t *testing.T) {
	t.Parallel()

	if _, err := editor.New(editor.WithStderr(nil)); err == nil {
		t.Fatal("expected error for nil stderr, got nil")
	} else if got := err.Error(); !contains(got, "stderr cannot be nil") {
		t.Fatalf("expected error to contain %q, got %q", "stderr cannot be nil", got)
	}
}

func TestNewWithAllOptions(t *testing.T) {
	t.Parallel()

	e, err := editor.New(
		editor.WithCmd("nvim"),
		editor.WithStdin(os.Stdin),
		editor.WithStdout(io.Discard),
		editor.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := e.Cmd(); got != "nvim" {
		t.Fatalf("expected Cmd() == %q, got %q", "nvim", got)
	}
}

// ---------------------------------------------------------------------------
// Task 3: Edit method
// ---------------------------------------------------------------------------

func TestEditReturnsUnmodifiedContent(t *testing.T) {
	t.Parallel()

	e, err := editor.New(
		editor.WithCmd("true"),
		editor.WithStdout(io.Discard),
		editor.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatalf("unexpected error creating editor: %v", err)
	}

	input := []byte("hello: world\n")

	got, err := e.Edit(input)
	if err != nil {
		t.Fatalf("unexpected error from Edit: %v", err)
	}

	if string(got) != string(input) {
		t.Fatalf("expected content %q, got %q", string(input), string(got))
	}
}

func TestEditReturnsModifiedContent(t *testing.T) {
	t.Parallel()

	// Create a source file with the desired modified content.
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "modified.yaml")

	want := "modified content"

	if err := os.WriteFile(srcFile, []byte(want), 0o600); err != nil {
		t.Fatalf("cannot write source file: %v", err)
	}

	// "cp <srcfile>" will receive the temp file path as the last argument,
	// resulting in: cp <srcfile> <tmpfile>, which copies source over temp.
	e, err := editor.New(
		editor.WithCmd("cp "+srcFile),
		editor.WithStdout(io.Discard),
		editor.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatalf("unexpected error creating editor: %v", err)
	}

	got, err := e.Edit([]byte("original content"))
	if err != nil {
		t.Fatalf("unexpected error from Edit: %v", err)
	}

	if string(got) != want {
		t.Fatalf("expected content %q, got %q", want, string(got))
	}
}

func TestEditEmptyContent(t *testing.T) {
	t.Parallel()

	e, err := editor.New(
		editor.WithCmd("true"),
		editor.WithStdout(io.Discard),
		editor.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatalf("unexpected error creating editor: %v", err)
	}

	got, err := e.Edit([]byte{})
	if err != nil {
		t.Fatalf("unexpected error from Edit: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty content, got %q", string(got))
	}
}

func TestEditFailsWhenEditorExitsNonZero(t *testing.T) {
	t.Parallel()

	e, err := editor.New(
		editor.WithCmd("false"),
		editor.WithStdout(io.Discard),
		editor.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatalf("unexpected error creating editor: %v", err)
	}

	if _, err = e.Edit([]byte("some content")); err == nil {
		t.Fatal("expected error when editor exits non-zero, got nil")
	} else if got := err.Error(); !contains(got, "editor failed") {
		t.Fatalf("expected error to contain %q, got %q", "editor failed", got)
	}
}

func TestEditFailsWhenCmdNotFound(t *testing.T) {
	t.Parallel()

	e, err := editor.New(
		editor.WithCmd("nonexistent-editor-xyz"),
		editor.WithStdout(io.Discard),
		editor.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatalf("unexpected error creating editor: %v", err)
	}

	if _, err = e.Edit([]byte("some content")); err == nil {
		t.Fatal("expected error when command not found, got nil")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
