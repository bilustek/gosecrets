package main

import (
	"bytes"
	"errors"
	"io"
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

func TestRunVersion(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"version", "--version", "-v"} {
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
// resolveEnv tests
// ---------------------------------------------------------------------------

func TestResolveEnvWithFlag(t *testing.T) {
	t.Parallel()

	args := []string{"init", "--env", "production"}
	env := resolveEnv(&args)

	if env != "production" {
		t.Fatalf("expected env %q, got %q", "production", env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveEnvWithEquals(t *testing.T) {
	t.Parallel()

	args := []string{"init", "--env=staging"}
	env := resolveEnv(&args)

	if env != "staging" {
		t.Fatalf("expected env %q, got %q", "staging", env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveEnvWithoutFlag(t *testing.T) {
	args := []string{"init"}

	t.Setenv(store.EnvEnv, "")

	env := resolveEnv(&args)

	if env != store.DefaultEnv {
		t.Fatalf("expected env %q, got %q", store.DefaultEnv, env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveEnvFromEnvVar(t *testing.T) {
	args := []string{"init"}

	t.Setenv(store.EnvEnv, "staging")

	env := resolveEnv(&args)

	if env != "staging" {
		t.Fatalf("expected env %q, got %q", "staging", env)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveEnvAtBeginning(t *testing.T) {
	t.Parallel()

	args := []string{"--env", "test", "show"}
	env := resolveEnv(&args)

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

// captureStdout captures stdout written during fn and returns it as a string.
// NOT safe for parallel tests.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}

// setupStore initializes a store in dir with content and sets the master key env var.
// NOT safe for parallel tests (uses t.Setenv).
func setupStore(t *testing.T, dir, env string, content []byte) {
	t.Helper()

	opts := []store.Option{store.WithEnv(env), store.WithRoot(dir)}

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

	if err := cmdInit(store.DefaultEnv, ""); err != nil {
		t.Fatalf("cmdInit() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "development.key")); err != nil {
		t.Fatalf("development.key should exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join("secrets", "development.enc")); err != nil {
		t.Fatalf("development.enc should exist: %v", err)
	}
}

func TestCmdInitWithEnv(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	if err := cmdInit("production", ""); err != nil {
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

	if err := cmdInit(store.DefaultEnv, ""); err != nil {
		t.Fatal(err)
	}

	if err := cmdInit(store.DefaultEnv, ""); err == nil {
		t.Fatal("expected error on second init, got nil")
	} else if !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("expected error to contain %q, got: %v", "already initialized", err)
	}
}

func TestCmdInitNextStepsDefault(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	out := captureStdout(t, func() {
		if err := cmdInit(store.DefaultEnv, ""); err != nil {
			t.Fatalf("cmdInit() error = %v", err)
		}
	})

	want := "gosecrets edit    # add your secrets"
	if !strings.Contains(out, want) {
		t.Fatalf("expected next steps to contain %q, got:\n%s", want, out)
	}

	noWant := "gosecrets edit --env"
	if strings.Contains(out, noWant) {
		t.Fatalf("default env should not contain %q, got:\n%s", noWant, out)
	}
}

func TestCmdInitNextStepsWithEnv(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	out := captureStdout(t, func() {
		if err := cmdInit("production", ""); err != nil {
			t.Fatalf("cmdInit(production) error = %v", err)
		}
	})

	want := "gosecrets edit --env production    # add your secrets"
	if !strings.Contains(out, want) {
		t.Fatalf("expected next steps to contain %q, got:\n%s", want, out)
	}
}

// ---------------------------------------------------------------------------
// cmdShow tests (NOT parallel — uses os.Chdir + t.Setenv)
// ---------------------------------------------------------------------------

func TestCmdShow(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: show-test\n"))

	if err := cmdShow(store.DefaultEnv, ""); err != nil {
		t.Fatalf("cmdShow() error = %v", err)
	}
}

func TestCmdShowFailsWithoutKey(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	t.Setenv(store.EnvMasterKey, "")

	if err := cmdShow(store.DefaultEnv, ""); err == nil {
		t.Fatal("expected error when master key is missing, got nil")
	}
}

// ---------------------------------------------------------------------------
// cmdGet tests (NOT parallel — uses os.Chdir + t.Setenv)
// ---------------------------------------------------------------------------

func TestCmdGet(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: get-test-123\ndatabase:\n  host: localhost\n"))

	if err := cmdGet("api_key", store.DefaultEnv, ""); err != nil {
		t.Fatalf("cmdGet(api_key) error = %v", err)
	}
}

func TestCmdGetDotNotation(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("database:\n  host: localhost\n"))

	if err := cmdGet("database.host", store.DefaultEnv, ""); err != nil {
		t.Fatalf("cmdGet(database.host) error = %v", err)
	}
}

func TestCmdGetMissingKey(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: value\n"))

	if err := cmdGet("nonexistent", store.DefaultEnv, ""); err == nil {
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

// ---------------------------------------------------------------------------
// cmdCompleteKeys tests (NOT parallel — uses os.Chdir + t.Setenv)
// ---------------------------------------------------------------------------

func TestCmdCompleteKeys(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: test\ndatabase:\n  host: localhost\n"))

	out := captureStdout(t, func() {
		if err := cmdCompleteKeys(store.DefaultEnv, ""); err != nil {
			t.Fatalf("cmdCompleteKeys() error = %v", err)
		}
	})

	for _, want := range []string{"api_key", "database.host"} {
		if !strings.Contains(out, want) {
			t.Errorf("cmdCompleteKeys() output missing %q, got:\n%s", want, out)
		}
	}
}

func TestCmdCompleteKeysSilentOnError(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	t.Setenv(store.EnvMasterKey, "")

	// no store initialized — should return nil (no error), empty output
	if err := cmdCompleteKeys(store.DefaultEnv, ""); err != nil {
		t.Fatalf("cmdCompleteKeys() should not error during completion, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// cmdCompletion tests (NOT parallel — captureStdout mutates os.Stdout)
// ---------------------------------------------------------------------------

func TestCmdCompletionBash(t *testing.T) {
	out := captureStdout(t, func() {
		if err := cmdCompletion("bash"); err != nil {
			t.Fatalf("cmdCompletion(bash) error = %v", err)
		}
	})

	if !strings.Contains(out, "_gosecrets") {
		t.Errorf("bash completion should contain _gosecrets function, got:\n%s", out[:200])
	}

	if !strings.Contains(out, "complete -F _gosecrets gosecrets") {
		t.Error("bash completion should register complete command")
	}
}

func TestCmdCompletionUnsupportedShell(t *testing.T) {
	if err := cmdCompletion("fish"); err == nil {
		t.Fatal("expected error for unsupported shell, got nil")
	} else if !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCompletion(t *testing.T) {
	if err := run([]string{"completion", "bash"}); err != nil {
		t.Fatalf("run(completion bash) error = %v", err)
	}
}

func TestRunCompletionNoArgs(t *testing.T) {
	if err := run([]string{"completion"}); err == nil {
		t.Fatal("expected error for completion without shell arg, got nil")
	}
}

func TestRunCompleteKeys(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: test\n"))

	if err := run([]string{"__complete-keys"}); err != nil {
		t.Fatalf("run(__complete-keys) error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// resolveRoot tests
// ---------------------------------------------------------------------------

func TestResolveRootWithFlag(t *testing.T) {
	t.Parallel()

	args := []string{"init", "--root", "/app"}
	root := resolveRoot(&args)

	if root != "/app" {
		t.Fatalf("expected root %q, got %q", "/app", root)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveRootWithEquals(t *testing.T) {
	t.Parallel()

	args := []string{"init", "--root=/opt/myapp"}
	root := resolveRoot(&args)

	if root != "/opt/myapp" {
		t.Fatalf("expected root %q, got %q", "/opt/myapp", root)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveRootWithoutFlag(t *testing.T) {
	args := []string{"init"}

	t.Setenv(store.EnvRoot, "")

	root := resolveRoot(&args)

	if root != "" {
		t.Fatalf("expected empty root, got %q", root)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveRootFromEnvVar(t *testing.T) {
	args := []string{"init"}

	t.Setenv(store.EnvRoot, "/from/env")

	root := resolveRoot(&args)

	if root != "/from/env" {
		t.Fatalf("expected root %q, got %q", "/from/env", root)
	}

	if len(args) != 1 || args[0] != "init" {
		t.Fatalf("expected args [init], got %v", args)
	}
}

func TestResolveRootFlagOverridesEnvVar(t *testing.T) {
	args := []string{"init", "--root", "/from/flag"}

	t.Setenv(store.EnvRoot, "/from/env")

	root := resolveRoot(&args)

	if root != "/from/flag" {
		t.Fatalf("expected root %q, got %q", "/from/flag", root)
	}
}

func TestResolveRootAtBeginning(t *testing.T) {
	t.Parallel()

	args := []string{"--root", "/app", "show"}
	root := resolveRoot(&args)

	if root != "/app" {
		t.Fatalf("expected root %q, got %q", "/app", root)
	}

	if len(args) != 1 || args[0] != "show" {
		t.Fatalf("expected args [show], got %v", args)
	}
}

// ---------------------------------------------------------------------------
// --root integration tests (NOT parallel — uses os.Chdir / t.Setenv)
// ---------------------------------------------------------------------------

func TestCmdInitWithRoot(t *testing.T) {
	dir := t.TempDir()

	if err := cmdInit(store.DefaultEnv, dir); err != nil {
		t.Fatalf("cmdInit(root=%s) error = %v", dir, err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "development.key")); err != nil {
		t.Fatalf("development.key should exist at root: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "development.enc")); err != nil {
		t.Fatalf("development.enc should exist at root: %v", err)
	}
}

func TestCmdInitWithRootAndEnv(t *testing.T) {
	dir := t.TempDir()

	if err := cmdInit("production", dir); err != nil {
		t.Fatalf("cmdInit(production, root=%s) error = %v", dir, err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "production.key")); err != nil {
		t.Fatalf("production.key should exist at root: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "production.enc")); err != nil {
		t.Fatalf("production.enc should exist at root: %v", err)
	}
}

func TestCmdShowWithRoot(t *testing.T) {
	dir := t.TempDir()
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: root-show-test\n"))

	if err := cmdShow(store.DefaultEnv, dir); err != nil {
		t.Fatalf("cmdShow(root=%s) error = %v", dir, err)
	}
}

func TestCmdGetWithRoot(t *testing.T) {
	dir := t.TempDir()
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: root-get-test\n"))

	if err := cmdGet("api_key", store.DefaultEnv, dir); err != nil {
		t.Fatalf("cmdGet(root=%s) error = %v", dir, err)
	}
}

func TestRunInitWithRootFlag(t *testing.T) {
	dir := t.TempDir()

	if err := run([]string{"init", "--root", dir}); err != nil {
		t.Fatalf("run(init --root) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "development.key")); err != nil {
		t.Fatalf("development.key should exist at root: %v", err)
	}
}

func TestRunInitWithRootEqualsFlag(t *testing.T) {
	dir := t.TempDir()

	if err := run([]string{"init", "--root=" + dir}); err != nil {
		t.Fatalf("run(init --root=dir) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "development.key")); err != nil {
		t.Fatalf("development.key should exist at root: %v", err)
	}
}

func TestRunInitWithRootAndEnvFlags(t *testing.T) {
	dir := t.TempDir()

	if err := run([]string{"init", "--root", dir, "--env", "staging"}); err != nil {
		t.Fatalf("run(init --root --env) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "staging.key")); err != nil {
		t.Fatalf("staging.key should exist at root: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "staging.enc")); err != nil {
		t.Fatalf("staging.enc should exist at root: %v", err)
	}
}

func TestRunInitWithRootEnvVar(t *testing.T) {
	dir := t.TempDir()

	t.Setenv(store.EnvRoot, dir)

	if err := run([]string{"init"}); err != nil {
		t.Fatalf("run(init) with GOSECRETS_ROOT error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets", "development.key")); err != nil {
		t.Fatalf("development.key should exist at GOSECRETS_ROOT: %v", err)
	}
}

func TestRunShowWithRootFlag(t *testing.T) {
	dir := t.TempDir()
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: test\n"))

	if err := run([]string{"show", "--root", dir}); err != nil {
		t.Fatalf("run(show --root) error = %v", err)
	}
}

func TestRunGetWithRootFlag(t *testing.T) {
	dir := t.TempDir()
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: root-test\n"))

	if err := run([]string{"get", "api_key", "--root", dir}); err != nil {
		t.Fatalf("run(get --root) error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// error path coverage tests (NOT parallel — uses t.Setenv)
// ---------------------------------------------------------------------------

func TestNewStoreErrorOnEmptyEnv(t *testing.T) {
	t.Parallel()

	if _, err := newStore("", ""); err == nil {
		t.Fatal("expected error for empty env, got nil")
	}
}

func TestCmdInitErrorOnEmptyEnv(t *testing.T) {
	t.Parallel()

	if err := cmdInit("", ""); err == nil {
		t.Fatal("expected error for empty env, got nil")
	}
}

func TestCmdShowErrorOnBadKey(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// init a store, then corrupt the master key env
	setupStore(t, dir, store.DefaultEnv, []byte("key: value\n"))
	t.Setenv(store.EnvMasterKey, "wrongkey")

	if err := cmdShow(store.DefaultEnv, ""); err == nil {
		t.Fatal("expected error for bad master key, got nil")
	}
}

func TestCmdGetErrorOnBadKey(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	setupStore(t, dir, store.DefaultEnv, []byte("key: value\n"))
	t.Setenv(store.EnvMasterKey, "wrongkey")

	if err := cmdGet("key", store.DefaultEnv, ""); err == nil {
		t.Fatal("expected error for bad master key, got nil")
	}
}

func TestCmdInitErrorOnNewStoreFailure(t *testing.T) {
	t.Parallel()

	if err := cmdInit("", "/some/path"); err == nil {
		t.Fatal("expected error for empty env in cmdInit, got nil")
	}
}

func TestCmdEditErrorOnNewStoreFailure(t *testing.T) {
	t.Parallel()

	if err := cmdEdit("", "/some/path"); err == nil {
		t.Fatal("expected error for empty env in cmdEdit, got nil")
	}
}

func TestCmdEditErrorOnMissingStore(t *testing.T) {
	dir := t.TempDir()

	t.Setenv(store.EnvMasterKey, "")

	if err := cmdEdit(store.DefaultEnv, dir); err == nil {
		t.Fatal("expected error for missing store in cmdEdit, got nil")
	}
}

func TestCmdGetVerifyOutput(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("api_key: hello-world\n"))

	out := captureStdout(t, func() {
		if err := cmdGet("api_key", store.DefaultEnv, ""); err != nil {
			t.Fatalf("cmdGet error = %v", err)
		}
	})

	if !strings.Contains(out, "hello-world") {
		t.Fatalf("expected output to contain %q, got: %s", "hello-world", out)
	}
}

func TestRunEditErrorOnMissingStore(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	t.Setenv(store.EnvMasterKey, "")

	if err := run([]string{"edit"}); err == nil {
		t.Fatal("expected error when store is missing, got nil")
	}
}

func TestCmdShowErrorOnNewStoreFailure(t *testing.T) {
	t.Parallel()

	if err := cmdShow("", ""); err == nil {
		t.Fatal("expected error for empty env in cmdShow, got nil")
	}
}

func TestCmdShowVerifyOutput(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)
	setupStore(t, dir, store.DefaultEnv, []byte("secret: show-me\n"))

	out := captureStdout(t, func() {
		if err := cmdShow(store.DefaultEnv, ""); err != nil {
			t.Fatalf("cmdShow error = %v", err)
		}
	})

	if !strings.Contains(out, "secret: show-me") {
		t.Fatalf("expected output to contain %q, got: %s", "secret: show-me", out)
	}
}
