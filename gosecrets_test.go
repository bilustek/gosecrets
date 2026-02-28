package gosecrets_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bilustek/gosecrets"
	"github.com/bilustek/gosecrets/internal/store"
)

func setupTestStore(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("database:\n  host: localhost\n  port: 5432\n  password: supersecret\n" +
		"api_key: sk-123\nenabled: true\ncount: 42\nnegative: -7\npi: 3.14\n" +
		"huge: 18446744073709551615\ntimeout: 5s\nretry_delay: 500ms\n")
	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	t.Setenv(store.EnvMasterKey, masterKey)

	return dir
}

// --- Load tests (NOT parallel — uses t.Setenv via setupTestStore) ---

func TestLoadReturnsSecrets(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.String("api_key")
	if got != "sk-123" {
		t.Errorf("String(api_key) = %q, want %q", got, "sk-123")
	}
}

func TestLoadWithEnv(t *testing.T) {
	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir), store.WithEnv("production"))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("api_key: prod-key-456\n")
	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	t.Setenv(store.EnvMasterKey, masterKey)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir), gosecrets.WithEnv("production"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.String("api_key")
	if got != "prod-key-456" {
		t.Errorf("String(api_key) = %q, want %q", got, "prod-key-456")
	}
}

func TestLoadFailsWhenKeyMissing(t *testing.T) {
	dir := t.TempDir()

	// Create the secrets directory but don't init (no key file)
	secretsDir := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	// Make sure env var is not set
	t.Setenv(store.EnvMasterKey, "")

	if _, err := gosecrets.Load(gosecrets.WithRoot(dir)); err == nil {
		t.Fatal("Load() expected error when key is missing, got nil")
	}
}

func TestLoadFailsWhenCredentialsMissing(t *testing.T) {
	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv(store.EnvMasterKey, masterKey)

	// Remove the credentials file
	credPath := s.CredentialsPath()
	if err = os.Remove(credPath); err != nil {
		t.Fatal(err)
	}

	if _, err = gosecrets.Load(gosecrets.WithRoot(dir)); err == nil {
		t.Fatal("Load() expected error when credentials file is missing, got nil")
	}
}

func TestLoadHandlesEmptyYAML(t *testing.T) {
	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	// Write empty content (valid YAML that produces nil map)
	if err = s.WriteCredentials([]byte(""), masterKey); err != nil {
		t.Fatal(err)
	}

	t.Setenv(store.EnvMasterKey, masterKey)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if secrets.All() == nil {
		t.Error("All() should return non-nil map for empty YAML")
	}
}

func TestLoadRejectsEmptyRoot(t *testing.T) {
	t.Parallel()

	if _, err := gosecrets.Load(gosecrets.WithRoot("")); err == nil {
		t.Fatal("Load() expected error for empty root, got nil")
	} else if !strings.Contains(err.Error(), "root directory cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsEmptyEnv(t *testing.T) {
	t.Parallel()

	if _, err := gosecrets.Load(gosecrets.WithEnv("")); err == nil {
		t.Fatal("Load() expected error for empty env, got nil")
	} else if !strings.Contains(err.Error(), "env cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFailsWithInvalidYAML(t *testing.T) {
	dir := t.TempDir()

	s, err := store.New(store.WithRoot(dir))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	// Write invalid YAML
	if err = s.WriteCredentials([]byte("{{invalid yaml:::"), masterKey); err != nil {
		t.Fatal(err)
	}

	t.Setenv(store.EnvMasterKey, masterKey)

	if _, err = gosecrets.Load(gosecrets.WithRoot(dir)); err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	} else if !strings.Contains(err.Error(), "invalid YAML") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAutoDetectsEnvFromEnvVar(t *testing.T) {
	dir := t.TempDir()

	// Create a "staging" store
	s, err := store.New(store.WithRoot(dir), store.WithEnv("staging"))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("api_key: staging-key-789\n")
	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	// Set GOSECRETS_ENV=staging so Load() picks it up automatically
	t.Setenv(store.EnvEnv, "staging")
	t.Setenv(store.EnvMasterKey, masterKey)

	// Load without WithEnv — should auto-detect "staging"
	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.String("api_key")
	if got != "staging-key-789" {
		t.Errorf("String(api_key) = %q, want %q", got, "staging-key-789")
	}
}

func TestLoadWithEnvOverridesEnvVar(t *testing.T) {
	dir := t.TempDir()

	// Create a "production" store
	s, err := store.New(store.WithRoot(dir), store.WithEnv("production"))
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("api_key: prod-override\n")
	if err = s.WriteCredentials(content, masterKey); err != nil {
		t.Fatal(err)
	}

	// Set GOSECRETS_ENV=staging but explicitly pass WithEnv("production")
	t.Setenv(store.EnvEnv, "staging")
	t.Setenv(store.EnvMasterKey, masterKey)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir), gosecrets.WithEnv("production"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.String("api_key")
	if got != "prod-override" {
		t.Errorf("String(api_key) = %q, want %q", got, "prod-override")
	}
}

// --- Accessor tests (NOT parallel — uses t.Setenv via setupTestStore) ---

func TestGetDotNotation(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	host := secrets.String("database.host")
	if host != "localhost" {
		t.Errorf("String(database.host) = %q, want %q", host, "localhost")
	}

	password := secrets.String("database.password")
	if password != "supersecret" {
		t.Errorf("String(database.password) = %q, want %q", password, "supersecret")
	}
}

func TestGetReturnsNilForMissingKey(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if v := secrets.Get("nonexistent"); v != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", v)
	}

	if v := secrets.Get("database.nonexistent"); v != nil {
		t.Errorf("Get(database.nonexistent) = %v, want nil", v)
	}

	if v := secrets.Get("database.host.deep"); v != nil {
		t.Errorf("Get(database.host.deep) = %v, want nil", v)
	}
}

func TestInt(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := secrets.Int("database.port"); got != 5432 {
		t.Errorf("Int(database.port) = %d, want %d", got, 5432)
	}

	if got := secrets.Int("count"); got != 42 {
		t.Errorf("Int(count) = %d, want %d", got, 42)
	}

	if got := secrets.Int("nonexistent"); got != 0 {
		t.Errorf("Int(nonexistent) = %d, want %d", got, 0)
	}

	if got := secrets.Int("api_key"); got != 0 {
		t.Errorf("Int(api_key) = %d, want %d (string value should return 0)", got, 0)
	}

	if got := secrets.Int("negative"); got != -7 {
		t.Errorf("Int(negative) = %d, want %d", got, -7)
	}

	if got := secrets.Int("pi"); got != 3 {
		t.Errorf("Int(pi) = %d, want %d (float truncated)", got, 3)
	}

	if got := secrets.Int("huge"); got != 0 {
		t.Errorf("Int(huge) = %d, want 0 (overflow should return 0)", got)
	}
}

func TestInt64(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := secrets.Int64("count"); got != 42 {
		t.Errorf("Int64(count) = %d, want %d", got, 42)
	}

	if got := secrets.Int64("negative"); got != -7 {
		t.Errorf("Int64(negative) = %d, want %d", got, -7)
	}

	if got := secrets.Int64("pi"); got != 3 {
		t.Errorf("Int64(pi) = %d, want %d (float truncated)", got, 3)
	}

	if got := secrets.Int64("huge"); got != 0 {
		t.Errorf("Int64(huge) = %d, want 0 (overflow should return 0)", got)
	}

	if got := secrets.Int64("nonexistent"); got != 0 {
		t.Errorf("Int64(nonexistent) = %d, want %d", got, 0)
	}

	if got := secrets.Int64("api_key"); got != 0 {
		t.Errorf("Int64(api_key) = %d, want %d (string should return 0)", got, 0)
	}
}

func TestFloat64(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := secrets.Float64("pi"); got != 3.14 {
		t.Errorf("Float64(pi) = %f, want %f", got, 3.14)
	}

	if got := secrets.Float64("count"); got != 42.0 {
		t.Errorf("Float64(count) = %f, want %f", got, 42.0)
	}

	if got := secrets.Float64("negative"); got != -7.0 {
		t.Errorf("Float64(negative) = %f, want %f", got, -7.0)
	}

	if got := secrets.Float64("nonexistent"); got != 0 {
		t.Errorf("Float64(nonexistent) = %f, want %f", got, 0.0)
	}

	if got := secrets.Float64("api_key"); got != 0 {
		t.Errorf("Float64(api_key) = %f, want %f (string should return 0)", got, 0.0)
	}
}

func TestDuration(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := secrets.Duration("timeout"); got != 5*time.Second {
		t.Errorf("Duration(timeout) = %v, want %v", got, 5*time.Second)
	}

	if got := secrets.Duration("retry_delay"); got != 500*time.Millisecond {
		t.Errorf("Duration(retry_delay) = %v, want %v", got, 500*time.Millisecond)
	}

	if got := secrets.Duration("nonexistent"); got != 0 {
		t.Errorf("Duration(nonexistent) = %v, want %v", got, time.Duration(0))
	}

	if got := secrets.Duration("count"); got != 0 {
		t.Errorf("Duration(count) = %v, want %v (non-string should return 0)", got, time.Duration(0))
	}

	if got := secrets.Duration("api_key"); got != 0 {
		t.Errorf("Duration(api_key) = %v, want %v (invalid duration string should return 0)", got, time.Duration(0))
	}
}

func TestBool(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := secrets.Bool("enabled"); !got {
		t.Error("Bool(enabled) = false, want true")
	}

	if got := secrets.Bool("nonexistent"); got {
		t.Error("Bool(nonexistent) = true, want false")
	}

	if got := secrets.Bool("api_key"); got {
		t.Error("Bool(api_key) = true, want false (string value should return false)")
	}
}

func TestMap(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	dbMap := secrets.Map("database")
	if dbMap == nil {
		t.Fatal("Map(database) = nil, want non-nil map")
	}

	if host, ok := dbMap["host"]; !ok || host != "localhost" {
		t.Errorf("Map(database)[host] = %v, want %q", host, "localhost")
	}

	if got := secrets.Map("nonexistent"); got != nil {
		t.Errorf("Map(nonexistent) = %v, want nil", got)
	}

	if got := secrets.Map("api_key"); got != nil {
		t.Errorf("Map(api_key) = %v, want nil (scalar value should return nil)", got)
	}
}

func TestHas(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !secrets.Has("api_key") {
		t.Error("Has(api_key) = false, want true")
	}

	if !secrets.Has("database.host") {
		t.Error("Has(database.host) = false, want true")
	}

	if secrets.Has("nonexistent") {
		t.Error("Has(nonexistent) = true, want false")
	}
}

func TestAll(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	all := secrets.All()
	if all == nil {
		t.Fatal("All() = nil, want non-nil map")
	}

	if _, ok := all["api_key"]; !ok {
		t.Error("All() does not contain api_key")
	}
}

func TestStringFormatsNonStringValues(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.String("database.port")
	if got != "5432" {
		t.Errorf("String(database.port) = %q, want %q", got, "5432")
	}

	got = secrets.String("nonexistent")
	if got != "" {
		t.Errorf("String(nonexistent) = %q, want %q", got, "")
	}
}

// --- Must* tests (NOT parallel — uses t.Setenv via setupTestStore) ---

func TestMustGetReturnsValue(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.MustGet("api_key")
	if got != "sk-123" {
		t.Errorf("MustGet(api_key) = %v, want %q", got, "sk-123")
	}
}

func TestMustGetPanicsForMissingKey(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustGet(nonexistent) did not panic")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %v", r)
		}

		if !strings.Contains(msg, "nonexistent") {
			t.Errorf("panic message %q does not contain key name %q", msg, "nonexistent")
		}
	}()

	secrets.MustGet("nonexistent")
}

func TestMustStringReturnsValue(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := secrets.MustString("api_key")
	if got != "sk-123" {
		t.Errorf("MustString(api_key) = %q, want %q", got, "sk-123")
	}
}

func TestMustStringPanicsForMissingKey(t *testing.T) {
	dir := setupTestStore(t)

	secrets, err := gosecrets.Load(gosecrets.WithRoot(dir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustString(nonexistent) did not panic")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %v", r)
		}

		if !strings.Contains(msg, "nonexistent") {
			t.Errorf("panic message %q does not contain key name %q", msg, "nonexistent")
		}
	}()

	secrets.MustString("nonexistent")
}
