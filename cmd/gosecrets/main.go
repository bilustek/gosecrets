package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bilustek/gosecrets"
	"github.com/bilustek/gosecrets/internal/editor"
	"github.com/bilustek/gosecrets/internal/store"
)

const usage = `gosecrets - encrypted credentials for Go projects

Usage:
  gosecrets init [--env ENV]       Initialize a new credential store
  gosecrets edit [--env ENV]       Edit credentials in $EDITOR
  gosecrets show [--env ENV]       Print decrypted credentials to stdout
  gosecrets get KEY [--env ENV]    Get a specific value (dot notation)
  gosecrets help                   Show this help

Environment:
  GOSECRETS_ENV                    Environment name (default: development)
  GOSECRETS_MASTER_KEY             Master key (overrides all key files)
  GOSECRETS_<ENV>_KEY              Environment-specific key (e.g. GOSECRETS_PRODUCTION_KEY)
  EDITOR / VISUAL                  Preferred text editor

Examples:
  gosecrets init                   Creates secrets/development.key + secrets/development.enc
  gosecrets init --env production  Creates secrets/production.key + secrets/production.enc
  gosecrets edit                   Opens credentials in your editor
  gosecrets get database.password  Prints a specific value
`

var errUnknownCommand = errors.New("unknown command")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)

		return nil
	}

	env := resolveEnv(&args)

	if len(args) == 0 {
		fmt.Print(usage)

		return nil
	}

	switch args[0] {
	case "init":
		return cmdInit(env)
	case "edit":
		return cmdEdit(env)
	case "show":
		return cmdShow(env)
	case "get":
		if len(args) < 2 {
			return errors.New("usage: gosecrets get KEY [--env ENV]")
		}

		return cmdGet(args[1], env)
	case "help", "--help", "-h":
		fmt.Print(usage)

		return nil
	default:
		return fmt.Errorf("%w: %q", errUnknownCommand, args[0])
	}
}

func cmdInit(env string) error {
	s, err := newStore(env)
	if err != nil {
		return err
	}

	masterKey, err := s.Init()
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	fmt.Println("credential store initialized")
	fmt.Println()
	fmt.Printf("  credentials: %s\n", s.CredentialsPath())
	fmt.Printf("  master key:  %s\n", s.KeyPath())
	fmt.Println()
	fmt.Println("add this to your .gitignore:")
	fmt.Printf("  %s\n", s.KeyFile())
	fmt.Println()
	fmt.Printf("  master key: %s\n", masterKey)
	fmt.Println("  save this key somewhere safe, you need it to decrypt your credentials.")
	fmt.Println()
	fmt.Println("next steps:")
	fmt.Println("  gosecrets edit    # add your secrets")

	return nil
}

func cmdEdit(env string) error {
	s, err := newStore(env)
	if err != nil {
		return err
	}

	masterKey, err := s.MasterKey()
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	plaintext, err := s.ReadCredentials(masterKey)
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	e, err := editor.New()
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	modified, err := e.Edit(plaintext)
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	if err = s.WriteCredentials(modified, masterKey); err != nil {
		return fmt.Errorf("edit: %w", err)
	}

	fmt.Println("credentials saved")

	return nil
}

func cmdShow(env string) error {
	s, err := newStore(env)
	if err != nil {
		return err
	}

	masterKey, err := s.MasterKey()
	if err != nil {
		return fmt.Errorf("show: %w", err)
	}

	plaintext, err := s.ReadCredentials(masterKey)
	if err != nil {
		return fmt.Errorf("show: %w", err)
	}

	fmt.Print(string(plaintext))

	return nil
}

func cmdGet(key, env string) error {
	secrets, err := gosecrets.Load(buildLoadOpts(env)...)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}

	if !secrets.Has(key) {
		return fmt.Errorf("get: key %q not found", key)
	}

	fmt.Print(secrets.String(key))

	return nil
}

func newStore(env string) (*store.Store, error) {
	s, err := store.New(buildStoreOpts(env)...)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	return s, nil
}

func buildStoreOpts(env string) []store.Option {
	return []store.Option{store.WithEnv(env)}
}

func buildLoadOpts(env string) []gosecrets.Option {
	return []gosecrets.Option{gosecrets.WithEnv(env)}
}

func resolveEnv(args *[]string) string {
	for i, arg := range *args {
		if arg == "--env" && i+1 < len(*args) {
			env := (*args)[i+1]
			*args = append((*args)[:i], (*args)[i+2:]...)

			return env
		}

		if strings.HasPrefix(arg, "--env=") {
			env := strings.TrimPrefix(arg, "--env=")
			*args = append((*args)[:i], (*args)[i+1:]...)

			return env
		}
	}

	if env := os.Getenv(store.EnvEnv); env != "" {
		return env
	}

	return store.DefaultEnv
}
