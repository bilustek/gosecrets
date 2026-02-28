package store_test

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/bilustek/gosecrets/internal/krypto"
	"github.com/bilustek/gosecrets/internal/store"
)

// newTestStore creates a Store rooted in a temporary directory for testing.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir))
	if err != nil {
		t.Fatal(err)
	}

	return s
}

// ---------------------------------------------------------------------------
// Task 4: Tests for New constructor and options
// ---------------------------------------------------------------------------

func TestNewReturnsDefaultStore(t *testing.T) {
	t.Parallel()

	s, err := store.New()
	if err != nil {
		t.Fatal(err)
	}

	if s.Dir() != "secrets" {
		t.Fatalf("expected default dir %q, got %q", "secrets", s.Dir())
	}

	if s.CredentialsFile() != "development.enc" {
		t.Fatalf("expected default credentials file %q, got %q", "development.enc", s.CredentialsFile())
	}

	if s.KeyFile() != "development.key" {
		t.Fatalf("expected default key file %q, got %q", "development.key", s.KeyFile())
	}
}

func TestNewWithRoot(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.WithRoot("/app"))
	if err != nil {
		t.Fatal(err)
	}

	want := "/app/secrets"
	if s.Dir() != want {
		t.Fatalf("expected dir %q, got %q", want, s.Dir())
	}
}

func TestNewWithEnv(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.WithEnv("staging"))
	if err != nil {
		t.Fatal(err)
	}

	if s.CredentialsFile() != "staging.enc" {
		t.Fatalf("expected credentials file %q, got %q", "staging.enc", s.CredentialsFile())
	}

	if s.KeyFile() != "staging.key" {
		t.Fatalf("expected key file %q, got %q", "staging.key", s.KeyFile())
	}
}

func TestNewWithRootAndEnv(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.WithRoot("/myapp"), store.WithEnv("production"))
	if err != nil {
		t.Fatal(err)
	}

	wantDir := "/myapp/secrets"
	if s.Dir() != wantDir {
		t.Fatalf("expected dir %q, got %q", wantDir, s.Dir())
	}

	if s.CredentialsFile() != "production.enc" {
		t.Fatalf("expected credentials file %q, got %q", "production.enc", s.CredentialsFile())
	}

	if s.KeyFile() != "production.key" {
		t.Fatalf("expected key file %q, got %q", "production.key", s.KeyFile())
	}
}

func TestNewRejectsEmptyRoot(t *testing.T) {
	t.Parallel()

	if _, err := store.New(store.WithRoot("")); err == nil {
		t.Fatal("expected error for empty root, got nil")
	} else if !strings.Contains(err.Error(), "root directory cannot be empty") {
		t.Fatalf("expected error to contain %q, got: %v", "root directory cannot be empty", err)
	}
}

func TestNewRejectsEmptyEnv(t *testing.T) {
	t.Parallel()

	if _, err := store.New(store.WithEnv("")); err == nil {
		t.Fatal("expected error for empty env, got nil")
	} else if !strings.Contains(err.Error(), "env cannot be empty") {
		t.Fatalf("expected error to contain %q, got: %v", "env cannot be empty", err)
	}
}

// ---------------------------------------------------------------------------
// Task 5: Tests for Init and MasterKey
// ---------------------------------------------------------------------------

func TestInitCreatesFiles(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	// Verify master key is 64 hex characters (32 bytes).
	if len(masterKey) != 64 {
		t.Fatalf("expected master key length 64, got %d", len(masterKey))
	}

	if _, err = hex.DecodeString(masterKey); err != nil {
		t.Fatalf("master key is not valid hex: %v", err)
	}

	// Verify key file exists.
	if _, err = os.Stat(s.KeyPath()); err != nil {
		t.Fatalf("key file should exist at %s: %v", s.KeyPath(), err)
	}

	// Verify credentials file exists.
	if _, err = os.Stat(s.CredentialsPath()); err != nil {
		t.Fatalf("credentials file should exist at %s: %v", s.CredentialsPath(), err)
	}
}

func TestInitRejectsDoubleInit(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	if _, err := s.Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Init(); err == nil {
		t.Fatal("expected error on second Init, got nil")
	} else if !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("expected error to contain %q, got: %v", "already initialized", err)
	}
}

func TestMasterKeyFromEnvVar(t *testing.T) {
	// NOT parallel: uses t.Setenv which modifies process environment.

	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir))
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Repeat("ab", 32)
	t.Setenv(store.EnvMasterKey, want)

	got, err := s.MasterKey()
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Fatalf("expected master key %q, got %q", want, got)
	}
}

func TestMasterKeyFromEnvSpecificVar(t *testing.T) {
	// NOT parallel: uses t.Setenv which modifies process environment.

	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir), store.WithEnv("production"))
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Repeat("cd", 32)
	t.Setenv("GOSECRETS_PRODUCTION_KEY", want)

	got, err := s.MasterKey()
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Fatalf("expected master key %q, got %q", want, got)
	}
}

func TestMasterKeyFromFile(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.MasterKey()
	if err != nil {
		t.Fatal(err)
	}

	if got != masterKey {
		t.Fatalf("expected master key %q from file, got %q", masterKey, got)
	}
}

func TestMasterKeyFailsWhenMissing(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	// No Init, no env var set.
	if _, err := s.MasterKey(); err == nil {
		t.Fatal("expected error when master key is missing, got nil")
	} else if !strings.Contains(err.Error(), "cannot read master key") {
		t.Fatalf("expected error to contain %q, got: %v", "cannot read master key", err)
	}
}

// ---------------------------------------------------------------------------
// Task 6: Tests for ReadCredentials, WriteCredentials, and path helpers
// ---------------------------------------------------------------------------

func TestWriteAndReadCredentials(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("database:\n  host: localhost\n  password: s3cret\napi_key: sk-test-123\n")

	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadCredentials(masterKey)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(content) {
		t.Fatalf("expected content %q, got %q", content, got)
	}
}

func TestReadCredentialsFailsWithWrongKey(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("secret: value\n")
	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	// Generate a different valid 64-char hex key.
	wrongKey, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// Ensure the wrong key is actually different.
	if wrongKey == masterKey {
		t.Fatal("wrong key should differ from master key")
	}

	if _, err = s.ReadCredentials(wrongKey); err == nil {
		t.Fatal("expected error when reading with wrong key, got nil")
	}
}

func TestReadCredentialsFailsWhenFileMissing(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	// No Init called, so credentials file does not exist.
	key, err := krypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	if _, err = s.ReadCredentials(key); err == nil {
		t.Fatal("expected error when credentials file is missing, got nil")
	} else if !strings.Contains(err.Error(), "cannot read credentials file") {
		t.Fatalf("expected error to contain %q, got: %v", "cannot read credentials file", err)
	}
}

func TestCredentialsPath(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.WithRoot("/app"))
	if err != nil {
		t.Fatal(err)
	}

	want := "/app/secrets/development.enc"
	if got := s.CredentialsPath(); got != want {
		t.Fatalf("expected credentials path %q, got %q", want, got)
	}
}

func TestKeyPath(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.WithRoot("/app"))
	if err != nil {
		t.Fatal(err)
	}

	want := "/app/secrets/development.key"
	if got := s.KeyPath(); got != want {
		t.Fatalf("expected key path %q, got %q", want, got)
	}
}
