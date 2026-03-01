// Package gosecrets provides encrypted credential management for Go projects,
// inspired by Rails credentials.
//
// Quick start:
//
//	// Initialize (run once)
//	gosecrets init
//
//	// Edit credentials
//	gosecrets edit
//
//	// In your Go code:
//	secrets, err := gosecrets.Load()
//	dbPass := secrets.String("database.password")
//	apiKey := secrets.MustString("api_key")
package gosecrets

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bilustek/gosecrets/internal/store"
	"github.com/goccy/go-yaml"
)

// Secrets holds the decrypted credentials as a nested map.
type Secrets struct {
	data map[string]any
}

type config struct {
	env       string
	storeOpts []store.Option
}

// Option configures credential loading.
type Option func(*config) error

// WithRoot sets the root directory for the credential store.
// The store directory will be root/secrets.
func WithRoot(root string) Option {
	return func(c *config) error {
		if root == "" {
			return errors.New("root directory cannot be empty")
		}
		c.storeOpts = append(c.storeOpts, store.WithRoot(root))
		return nil
	}
}

// WithEnv sets environment-specific credential files.
// For example, WithEnv("production") reads production.enc with production.key.
// This overrides the GOSECRETS_ENV environment variable.
func WithEnv(env string) Option {
	return func(c *config) error {
		if env == "" {
			return errors.New("env cannot be empty")
		}
		c.env = env
		return nil
	}
}

// Load reads and decrypts the credentials.
// Environment resolution order: WithEnv() > GOSECRETS_ENV > "development".
//
//	secrets, err := gosecrets.Load()
//	secrets, err := gosecrets.Load(gosecrets.WithRoot("/app"), gosecrets.WithEnv("production"))
func Load(opts ...Option) (*Secrets, error) {
	cfg := &config{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("gosecrets: %w", err)
		}
	}

	env := cfg.env
	if env == "" {
		env = os.Getenv(store.EnvEnv)
	}

	if env == "" {
		env = store.DefaultEnv
	}

	storeOpts := append(cfg.storeOpts, store.WithEnv(env))

	s, err := store.New(storeOpts...)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	masterKey, err := s.MasterKey()
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	plaintext, err := s.ReadCredentials(masterKey)
	if err != nil {
		return nil, fmt.Errorf("gosecrets: %w", err)
	}

	var data map[string]any
	if err = yaml.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("gosecrets: invalid YAML in credentials: %w", err)
	}

	if data == nil {
		data = make(map[string]any)
	}

	return &Secrets{data: data}, nil
}

// Get retrieves a value using dot notation (e.g., "database.password").
// Returns nil if the key doesn't exist.
func (s *Secrets) Get(key string) any {
	parts := strings.Split(key, ".")
	var current any = s.data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}

	return current
}

// String retrieves a string value using dot notation.
// Returns fallback (or empty string) if the key doesn't exist.
func (s *Secrets) String(key string, fallback ...string) string {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Int retrieves an integer value using dot notation.
// Returns fallback (or 0) if the key doesn't exist or isn't numeric.
func (s *Secrets) Int(key string, fallback ...int) int {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return 0
	}

	switch val := v.(type) {
	case uint64:
		if val > math.MaxInt {
			return 0
		}
		return int(val)
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

// Int64 retrieves an int64 value using dot notation.
// Returns fallback (or 0) if the key doesn't exist or isn't numeric.
func (s *Secrets) Int64(key string, fallback ...int64) int64 {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return 0
	}

	switch val := v.(type) {
	case int64:
		return val
	case uint64:
		if val > math.MaxInt64 {
			return 0
		}
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

// Float64 retrieves a float64 value using dot notation.
// Returns fallback (or 0) if the key doesn't exist or isn't numeric.
func (s *Secrets) Float64(key string, fallback ...float64) float64 {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return 0
	}

	switch val := v.(type) {
	case float64:
		return val
	case uint64:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// Duration retrieves a time.Duration value using dot notation.
// The value must be a string parseable by time.ParseDuration (e.g., "5s", "1h30m").
// Returns fallback (or 0) if the key doesn't exist, isn't a string, or can't be parsed.
func (s *Secrets) Duration(key string, fallback ...time.Duration) time.Duration {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return 0
	}

	str, ok := v.(string)
	if !ok {
		return 0
	}

	d, err := time.ParseDuration(str)
	if err != nil {
		return 0
	}

	return d
}

// Bool retrieves a boolean value using dot notation.
// Returns fallback (or false) if the key doesn't exist or isn't a bool.
func (s *Secrets) Bool(key string, fallback ...bool) bool {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return false
	}

	val, ok := v.(bool)
	if !ok {
		return false
	}

	return val
}

// Map retrieves a nested map using dot notation.
// Returns fallback (or nil) if the key doesn't exist or isn't a map.
func (s *Secrets) Map(key string, fallback ...map[string]any) map[string]any {
	v := s.Get(key)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}

		return nil
	}

	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}

	return m
}

// TCPAddr retrieves a TCP address value using dot notation.
// The value must be a string parseable by net.ResolveTCPAddr (e.g., "localhost:5432").
// Returns fallback parsed address (or nil) if the key doesn't exist or can't be resolved.
func (s *Secrets) TCPAddr(key string, fallback ...string) *net.TCPAddr {
	v := s.Get(key)

	var str string

	if v == nil {
		if len(fallback) == 0 {
			return nil
		}

		str = fallback[0]
	} else {
		switch val := v.(type) {
		case string:
			str = val
		default:
			str = fmt.Sprintf("%v", val)
		}
	}

	addr, err := net.ResolveTCPAddr("tcp", str)
	if err != nil {
		return nil
	}

	return addr
}

// MustTCPAddr is like TCPAddr but panics if the key doesn't exist or isn't a valid TCP address.
// Use this for required network addresses during application startup.
func (s *Secrets) MustTCPAddr(key string) *net.TCPAddr {
	v := s.Get(key)
	if v == nil {
		panic(fmt.Sprintf("gosecrets: required key %q not found", key))
	}

	var str string

	switch val := v.(type) {
	case string:
		str = val
	default:
		str = fmt.Sprintf("%v", val)
	}

	addr, err := net.ResolveTCPAddr("tcp", str)
	if err != nil {
		panic(fmt.Sprintf("gosecrets: key %q is not a valid TCP address: %v", key, err))
	}

	return addr
}

// MustGet is like Get but panics if the key doesn't exist.
// Use this for required credentials during application startup.
func (s *Secrets) MustGet(key string) any {
	v := s.Get(key)
	if v == nil {
		panic(fmt.Sprintf("gosecrets: required key %q not found", key))
	}

	return v
}

// MustString is like String but panics if the key doesn't exist.
// Use this for required credentials during application startup.
func (s *Secrets) MustString(key string) string {
	v := s.Get(key)
	if v == nil {
		panic(fmt.Sprintf("gosecrets: required key %q not found", key))
	}

	return s.String(key)
}

// Has checks if a key exists in the credentials.
func (s *Secrets) Has(key string) bool {
	return s.Get(key) != nil
}

// All returns the entire credentials map.
func (s *Secrets) All() map[string]any {
	return s.data
}
