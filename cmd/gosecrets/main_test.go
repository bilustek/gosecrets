package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bilustek/gosecrets/internal/store"
)

// ---------------------------------------------------------------------------
// run() routing tests (parallel — no filesystem or env mutation)
// ---------------------------------------------------------------------------

func TestRunNoArgs(t *testing.T) {
	t.Parallel()

	if err := run(nil); err != nil {
		t.Fatalf("run(nil) error = %v", err)
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"help", "--help", "-h"} {
		if err := run([]string{arg}); err != nil {
			t.Fatalf("run(%q) error = %v", arg, err)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	if err := run([]string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown command, got nil")
	} else if !errors.Is(err, errUnknownCommand) {
		t.Fatalf("expected errUnknownCommand, got: %v", err)
	}
}

func TestRunGetWithoutKey(t *testing.T) {
	t.Parallel()

	if err := run([]string{"get"}); err == nil {
		t.Fatal("expected error for get without key, got nil")
	}
}

func TestRunOnlyEnvFlag(t *testing.T) {
	t.Parallel()

	if err := run([]string{"--env", "prod"}); err != nil {
		t.Fatalf("run(--env prod) error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// extractEnv tests (parallel — pure function)
// ---------------------------------------------------------------------------

func TestExtractEnvWithFlag(t *testing.T) {
	t.Parallel()

	args := []string{"init", "--env", "production"}
	env := extractEnv(&args)

	if env != "production" {
		t.Fatalf("expected env %q, got %q", "production", env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestExtractEnvWithEquals(t *testing.T) {
	t.Parallel()

	args := []string{"init", "--env=staging"}
	env := extractEnv(&args)

	if env != "staging" {
		t.Fatalf("expected env %q, got %q", "staging", env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestExtractEnvWithoutFlag(t *testing.T) {
	t.Parallel()

	args := []string{"init"}
	env := extractEnv(&args)

	if env != "" {
		t.Fatalf("expected empty env, got %q", env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestExtractEnvAtBeginning(t *testing.T) {
	t.Parallel()

	args := []string{"--env", "test", "show"}
	env := extractEnv(&args)

	if env != "test" {
		t.Fatalf("expected env %q, got %q", "test", env)
	}

	if len(args) != 1 || args[0] != "show" {
		t.Fatalf("expected args [show], got %v", args)
	}
}

// ---------------------------------------------------------------------------
// helpers for command tests
// ---------------------------------------------------------------------------

// chdirTemp changes the working directory to dir and returns a cleanup func.
// NOT safe for parallel tests.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

// setupStore initializes a store in dir with content and sets the master key env var.
// NOT safe for parallel tests (uses t.Setenv).
func setupStore(t *testing.T, dir, env string, content []byte) {
	t.Helper()

	var opts []store.Option
	if env != "" {
		opts = append(opts, store.WithEnv(env))
	}

	opts = append(opts, store.WithRoot(dir))

	s, err := store.New(opts...)
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	t.Setenv(store.EnvMasterKey, masterKey)
}

// ---------------------------------------------------------------------------
// cmdInit tests (NOT parallel — uses os.Chdir)
// ---------------------------------------------------------------------------

func TestCmdInit(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	if err := cmdInit(""); err != nil {
		t.Fatalf("cmdInit() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "master.key")); err != nil {
		t.Fatalf("master.key should exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "credentials.enc")); err != nil {
		t.Fatalf("credentials.enc should exist: %v", err)
	}
}

func TestCmdInitWithEnv(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	if err := cmdInit("production"); err != nil {
		t.Fatalf("cmdInit(production) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "production.key")); err != nil {
		t.Fatalf("production.key should exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "production.enc")); err != nil {
		t.Fatalf("production.enc should exist: %v", err)
	}
}

func TestCmdInitRejectsDouble(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	if err := cmdInit(""); err != nil {
		t.Fatal(err)
	}

	if err := cmdInit(""); err == nil {
		t.Fatal("expected error on second init, got nil")
	} else if !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("expected error to contain %q, got: %v", "already initialized", err)
	}
}

// ---------------------------------------------------------------------------
// cmdShow tests (NOT parallel — uses os.Chdir + t.Setenv)
// ---------------------------------------------------------------------------

func TestCmdShow(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, "", []byte("api_key: show-test\n"))

	if err := cmdShow(""); err != nil {
		t.Fatalf("cmdShow() error = %v", err)
	}
}

func TestCmdShowFailsWithoutKey(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	t.Setenv(store.EnvMasterKey, "")

	if err := cmdShow(""); err == nil {
		t.Fatal("expected error when master key is missing, got nil")
	}
}

// ---------------------------------------------------------------------------
// cmdGet tests (NOT parallel — uses os.Chdir + t.Setenv)
// ---------------------------------------------------------------------------

func TestCmdGet(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, "", []byte("api_key: get-test-123\ndatabase:\n  host: localhost\n"))

	if err := cmdGet("api_key", ""); err != nil {
		t.Fatalf("cmdGet(api_key) error = %v", err)
	}
}

func TestCmdGetDotNotation(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, "", []byte("database:\n  host: localhost\n"))

	if err := cmdGet("database.host", ""); err != nil {
		t.Fatalf("cmdGet(database.host) error = %v", err)
	}
}

func TestCmdGetMissingKey(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, "", []byte("api_key: value\n"))

	if err := cmdGet("nonexistent", ""); err == nil {
		t.Fatal("expected error for missing key, got nil")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected error to contain %q, got: %v", "not found", err)
	}
}

// ---------------------------------------------------------------------------
// run() integration tests (NOT parallel — uses os.Chdir + t.Setenv)
// ---------------------------------------------------------------------------

func TestRunInitThenShowThenGet(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// init
	if err := run([]string{"init"}); err != nil {
		t.Fatalf("run(init) error = %v", err)
	}

	// show (needs master key — read from file, already created by init)
	if err := run([]string{"show"}); err != nil {
		t.Fatalf("run(show) error = %v", err)
	}
}

func TestRunInitWithEnvFlag(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	if err := run([]string{"init", "--env", "staging"}); err != nil {
		t.Fatalf("run(init --env staging) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "staging.key")); err != nil {
		t.Fatalf("staging.key should exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "staging.enc")); err != nil {
		t.Fatalf("staging.enc should exist: %v", err)
	}
}

func TestRunInitWithEnvEqualsFlag(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	if err := run([]string{"init", "--env=production"}); err != nil {
		t.Fatalf("run(init --env=production) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "production.key")); err != nil {
		t.Fatalf("production.key should exist: %v", err)
	}
}
