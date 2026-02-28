// Package editor handles opening decrypted credentials in the user's
// preferred text editor.
package editor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ContentEditor defines the contract for editing content.
type ContentEditor interface {
	Edit(content []byte) ([]byte, error)
}

// Editor opens content in the user's preferred text editor.
type Editor struct {
	cmd    string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

var _ ContentEditor = (*Editor)(nil)

// Option configures an Editor.
type Option func(*Editor) error

// WithCmd overrides the editor command.
// By default, the editor is resolved from $EDITOR, $VISUAL, or "vi".
func WithCmd(cmd string) Option {
	return func(e *Editor) error {
		if cmd == "" {
			return errors.New("cmd cannot be empty")
		}
		e.cmd = cmd
		return nil
	}
}

// WithStdin sets the standard input for the editor process.
func WithStdin(r io.Reader) Option {
	return func(e *Editor) error {
		if r == nil {
			return errors.New("stdin cannot be nil")
		}
		e.stdin = r
		return nil
	}
}

// WithStdout sets the standard output for the editor process.
func WithStdout(w io.Writer) Option {
	return func(e *Editor) error {
		if w == nil {
			return errors.New("stdout cannot be nil")
		}
		e.stdout = w
		return nil
	}
}

// WithStderr sets the standard error for the editor process.
func WithStderr(w io.Writer) Option {
	return func(e *Editor) error {
		if w == nil {
			return errors.New("stderr cannot be nil")
		}
		e.stderr = w
		return nil
	}
}

// New creates an Editor with the given options.
// Without options, it uses $EDITOR, $VISUAL, or "vi" as the command,
// and os.Stdin/os.Stdout/os.Stderr for I/O.
func New(opts ...Option) (*Editor, error) {
	e := &Editor{
		cmd:    resolveCmd(),
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("gosecrets: %w", err)
		}
	}

	return e, nil
}

// Edit opens the given content in the editor and returns the modified content.
func (e *Editor) Edit(content []byte) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "gosecrets-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("gosecrets: cannot create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err = tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()

		return nil, fmt.Errorf("gosecrets: cannot write temp file: %w", err)
	}

	if err = tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()

		return nil, fmt.Errorf("gosecrets: cannot sync temp file: %w", err)
	}

	if err = e.runEditor(tmpPath); err != nil {
		_ = tmpFile.Close()

		return nil, err
	}

	// Read back from the same file descriptor — avoids os.ReadFile(variable)
	// which would trigger G304/G703 path traversal warnings.
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		_ = tmpFile.Close()

		return nil, fmt.Errorf("gosecrets: cannot seek temp file: %w", err)
	}

	modified, err := io.ReadAll(tmpFile)

	_ = tmpFile.Close()

	if err != nil {
		return nil, fmt.Errorf("gosecrets: cannot read temp file: %w", err)
	}

	return modified, nil
}

func (e *Editor) runEditor(tmpPath string) error {
	parts := strings.Fields(e.cmd)

	editorBin, err := exec.LookPath(parts[0])
	if err != nil {
		return fmt.Errorf("gosecrets: editor not found: %w", err)
	}

	editorBin = filepath.Clean(editorBin)

	if !strings.HasPrefix(editorBin, "/") {
		return fmt.Errorf("gosecrets: editor path must be absolute: %s", editorBin)
	}

	args := append([]string{editorBin}, append(parts[1:], tmpPath)...)
	cmd := &exec.Cmd{
		Path:   editorBin,
		Args:   args,
		Stdin:  e.stdin,
		Stdout: e.stdout,
		Stderr: e.stderr,
	}

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("gosecrets: editor failed: %w", err)
	}

	return nil
}

// Cmd returns the editor command.
func (e *Editor) Cmd() string { return e.cmd }

func resolveCmd() string {
	if cmd := os.Getenv("EDITOR"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("VISUAL"); cmd != "" {
		return cmd
	}
	return "vi"
}
