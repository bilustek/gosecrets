// Package store manages encrypted credential files and master keys.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bilustek/gosecrets/internal/krypto"
)

const (
	// EnvMasterKey is the environment variable name for the master key.
	EnvMasterKey = "GOSECRETS_MASTER_KEY"

	defaultDir             = "secrets"
	defaultCredentialsFile = "credentials.enc"
	defaultKeyFile         = "master.key"

	permDir  = 0o750
	permFile = 0o600
)

// Store represents a gosecrets credential store.
type Store struct {
	dir             string
	credentialsFile string
	keyFile         string
}

// SecretStore defines the contract for credential storage operations.
type SecretStore interface {
	Init() (string, error)
	MasterKey() (string, error)
	ReadCredentials(masterKey string) ([]byte, error)
	WriteCredentials(content []byte, masterKey string) error
	CredentialsPath() string
	KeyPath() string
}

var _ SecretStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store) error

// WithRoot sets the root directory for the store.
// The store directory will be root/secrets.
func WithRoot(root string) Option {
	return func(s *Store) error {
		if root == "" {
			return errors.New("root directory cannot be empty")
		}
		s.dir = filepath.Join(root, defaultDir)
		return nil
	}
}

// WithEnv sets environment-specific file names.
// For example, WithEnv("production") uses production.enc and production.key.
func WithEnv(env string) Option {
	return func(s *Store) error {
		if env == "" {
			return errors.New("env cannot be empty")
		}
		s.credentialsFile = env + ".enc"
		s.keyFile = env + ".key"
		return nil
	}
}

// New creates a Store with the given options.
// Without options, it operates on ./secrets/ with default file names.
func New(opts ...Option) (*Store, error) {
	s := &Store{
		dir:             defaultDir,
		credentialsFile: defaultCredentialsFile,
		keyFile:         defaultKeyFile,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("gosecrets: %w", err)
		}
	}

	return s, nil
}

// Dir returns the store directory path.
func (s *Store) Dir() string { return s.dir }

// CredentialsFile returns the credentials filename.
func (s *Store) CredentialsFile() string { return s.credentialsFile }

// KeyFile returns the key filename.
func (s *Store) KeyFile() string { return s.keyFile }

// Init initializes a new credential store: creates the directory,
// generates a master key, and creates an empty encrypted credentials file.
// Returns the generated master key.
func (s *Store) Init() (string, error) {
	if err := os.MkdirAll(s.dir, permDir); err != nil {
		return "", fmt.Errorf("gosecrets: cannot create directory: %w", err)
	}

	// Check if already initialized
	keyPath := filepath.Join(s.dir, s.keyFile)
	if _, err := os.Stat(keyPath); err == nil {
		return "", fmt.Errorf("gosecrets: already initialized (%s exists)", s.keyFile)
	}

	// Generate master key
	masterKey, err := krypto.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("gosecrets: %w", err)
	}

	// Write master key file (readable only by owner)
	if err := os.WriteFile(keyPath, []byte(masterKey+"\n"), permFile); err != nil {
		return "", fmt.Errorf("gosecrets: cannot write key file: %w", err)
	}

	// Create empty credentials file with default content
	defaultContent := []byte("# Add your secrets here\n" +
		"# Example:\n" +
		"# database:\n" +
		"#   host: localhost\n" +
		"#   password: supersecret\n" +
		"# api_key: sk-123456\n")
	if err := s.WriteCredentials(defaultContent, masterKey); err != nil {
		// Cleanup key file on failure
		_ = os.Remove(keyPath)
		return "", err
	}

	return masterKey, nil
}

// MasterKey resolves the master key. Priority:
// 1. GOSECRETS_MASTER_KEY environment variable
// 2. Key file on disk
func (s *Store) MasterKey() (string, error) {
	// Check env var first
	if key := os.Getenv(EnvMasterKey); key != "" {
		return strings.TrimSpace(key), nil
	}

	// Check env-specific env var (e.g. GOSECRETS_PRODUCTION_KEY)
	if s.keyFile != defaultKeyFile {
		env := strings.TrimSuffix(s.keyFile, ".key")
		envVar := "GOSECRETS_" + strings.ToUpper(env) + "_KEY"
		if key := os.Getenv(envVar); key != "" {
			return strings.TrimSpace(key), nil
		}
	}

	// Read from file
	keyPath := filepath.Clean(filepath.Join(s.dir, s.keyFile))

	if !strings.HasPrefix(keyPath, filepath.Clean(s.dir)) {
		return "", fmt.Errorf("gosecrets: invalid key path: %s", keyPath)
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("gosecrets: cannot read master key (set %s or create %s): %w",
			EnvMasterKey, keyPath, err)
	}

	return strings.TrimSpace(string(data)), nil
}

// ReadCredentials decrypts and returns the raw credentials content.
func (s *Store) ReadCredentials(masterKey string) ([]byte, error) {
	credPath := s.CredentialsPath()
	credPath = filepath.Clean(credPath)

	if !strings.HasPrefix(credPath, filepath.Clean(s.dir)) {
		return nil, fmt.Errorf("gosecrets: invalid credentials path: %s", credPath)
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: cannot read credentials file: %w", err)
	}

	plaintext, err := krypto.Decrypt(data, masterKey)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	return plaintext, nil
}

// WriteCredentials encrypts and writes credentials content.
func (s *Store) WriteCredentials(content []byte, masterKey string) error {
	ciphertext, err := krypto.Encrypt(content, masterKey)
	if err != nil {
		return fmt.Errorf("gosecrets: %w", err)
	}

	credPath := filepath.Join(s.dir, s.credentialsFile)
	if err := os.WriteFile(credPath, ciphertext, permFile); err != nil {
		return fmt.Errorf("gosecrets: cannot write credentials file: %w", err)
	}

	return nil
}

// CredentialsPath returns the full path to the credentials file.
func (s *Store) CredentialsPath() string {
	return filepath.Join(s.dir, s.credentialsFile)
}

// KeyPath returns the full path to the key file.
func (s *Store) KeyPath() string {
	return filepath.Join(s.dir, s.keyFile)
}
