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

const version = "0.3.1"

const usage = `gosecrets - encrypted credentials for Go projects

Usage:
  gosecrets init [--env ENV] [--root DIR]       Initialize a new credential store
  gosecrets edit [--env ENV] [--root DIR]       Edit credentials in $EDITOR
  gosecrets show [--env ENV] [--root DIR]       Print decrypted credentials to stdout
  gosecrets get KEY [--env ENV] [--root DIR]    Get a specific value (dot notation)
  gosecrets version, --version, -v              Show version
  gosecrets help, --help, -h                    Show this help
  gosecrets completion bash                     Output bash completion script

Environment:
  GOSECRETS_ROOT                   Root directory for secrets/ (default: current directory)
  GOSECRETS_ENV                    Environment name (default: development)
  GOSECRETS_MASTER_KEY             Master key (overrides all key files)
  GOSECRETS_<ENV>_KEY              Environment-specific key (e.g. GOSECRETS_PRODUCTION_KEY)
  EDITOR / VISUAL                  Preferred text editor

Examples:
  gosecrets init                              Creates ./secrets/development.{key,enc}
  gosecrets init --env production             Creates ./secrets/production.{key,enc}
  gosecrets init --root /app --env production Creates /app/secrets/production.{key,enc}
  gosecrets edit                              Opens credentials in your editor
  gosecrets get database.password             Prints a specific value
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
	root := resolveRoot(&args)

	if len(args) == 0 {
		fmt.Print(usage)

		return nil
	}

	switch args[0] {
	case "init":
		return cmdInit(env, root)
	case "edit":
		return cmdEdit(env, root)
	case "show":
		return cmdShow(env, root)
	case "get":
		if len(args) < 2 {
			return errors.New("usage: gosecrets get KEY [--env ENV] [--root DIR]")
		}

		return cmdGet(args[1], env, root)
	case "version", "--version", "-v":
		fmt.Println(version)

		return nil
	case "completion":
		if len(args) < 2 {
			return errors.New("usage: gosecrets completion bash")
		}

		return cmdCompletion(args[1])
	case "__complete-keys":
		return cmdCompleteKeys(env, root)
	case "help", "--help", "-h":
		fmt.Print(usage)

		return nil
	default:
		return fmt.Errorf("%w: %q", errUnknownCommand, args[0])
	}
}

func cmdInit(env, root string) error {
	s, err := newStore(env, root)
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
	if env != store.DefaultEnv {
		fmt.Printf("  gosecrets edit --env %s    # add your secrets\n", env)
	} else {
		fmt.Println("  gosecrets edit    # add your secrets")
	}

	return nil
}

func cmdEdit(env, root string) error {
	s, err := newStore(env, root)
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

func cmdShow(env, root string) error {
	s, err := newStore(env, root)
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

func cmdGet(key, env, root string) error {
	secrets, err := gosecrets.Load(buildLoadOpts(env, root)...)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}

	if !secrets.Has(key) {
		return fmt.Errorf("get: key %q not found", key)
	}

	fmt.Println(secrets.String(key))

	return nil
}

func newStore(env, root string) (*store.Store, error) {
	s, err := store.New(buildStoreOpts(env, root)...)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	return s, nil
}

func buildStoreOpts(env, root string) []store.Option {
	opts := []store.Option{store.WithEnv(env)}
	if root != "" {
		opts = append(opts, store.WithRoot(root))
	}

	return opts
}

func buildLoadOpts(env, root string) []gosecrets.Option {
	opts := []gosecrets.Option{gosecrets.WithEnv(env)}
	if root != "" {
		opts = append(opts, gosecrets.WithRoot(root))
	}

	return opts
}

func cmdCompleteKeys(env, root string) error {
	secrets, err := gosecrets.Load(buildLoadOpts(env, root)...)
	if err != nil {
		return nil //nolint:nilerr // silence errors during completion
	}

	for _, key := range secrets.Keys() {
		fmt.Println(key)
	}

	return nil
}

func cmdCompletion(shell string) error {
	switch shell {
	case "bash":
		fmt.Print(bashCompletionScript)

		return nil
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash)", shell)
	}
}

const bashCompletionScript = `_gosecrets() {
    local cur prev words cword
    _init_completion || return

    local commands="init edit show get version help completion"

    # Find the subcommand position
    local subcmd=""
    local subcmd_idx=0
    local i
    for ((i=1; i < cword; i++)); do
        case "${words[i]}" in
            --env|--env=*|--root|--root=*)
                # skip --env/--root and their arguments
                if [[ "${words[i]}" == "--env" || "${words[i]}" == "--root" ]]; then
                    ((i++))
                fi
                ;;
            -*)
                ;;
            *)
                subcmd="${words[i]}"
                subcmd_idx=$i
                break
                ;;
        esac
    done

    # Complete --env value
    if [[ "$prev" == "--env" ]]; then
        local envs
        envs=$(find secrets -name '*.enc' -maxdepth 1 2>/dev/null | sed 's|secrets/||;s|\.enc$||')
        COMPREPLY=($(compgen -W "$envs" -- "$cur"))
        return
    fi

    # Complete --env= inline
    if [[ "$cur" == --env=* ]]; then
        local envs
        envs=$(find secrets -name '*.enc' -maxdepth 1 2>/dev/null | sed 's|secrets/||;s|\.enc$||')
        COMPREPLY=($(compgen -P "--env=" -W "$envs" -- "${cur#--env=}"))
        return
    fi

    # Complete --root value with directory names
    if [[ "$prev" == "--root" ]]; then
        COMPREPLY=($(compgen -d -- "$cur"))
        return
    fi

    # Complete --root= inline with directory names
    if [[ "$cur" == --root=* ]]; then
        COMPREPLY=($(compgen -P "--root=" -d -- "${cur#--root=}"))
        return
    fi

    # No subcommand yet — complete subcommands and flags
    if [[ -z "$subcmd" ]]; then
        if [[ "$cur" == -* ]]; then
            COMPREPLY=($(compgen -W "--env --root --version --help" -- "$cur"))
        else
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        fi
        return
    fi

    # Subcommand-specific completions
    case "$subcmd" in
        get)
            # Complete keys using hidden __complete-keys command
            if [[ $cword -eq $((subcmd_idx + 1)) ]]; then
                local env_arg=""
                local root_arg=""
                for ((i=1; i < ${#words[@]}; i++)); do
                    if [[ "${words[i]}" == "--env" && -n "${words[i+1]}" ]]; then
                        env_arg="--env ${words[i+1]}"
                    fi
                    if [[ "${words[i]}" == --env=* ]]; then
                        env_arg="--env ${words[i]#--env=}"
                    fi
                    if [[ "${words[i]}" == "--root" && -n "${words[i+1]}" ]]; then
                        root_arg="--root ${words[i+1]}"
                    fi
                    if [[ "${words[i]}" == --root=* ]]; then
                        root_arg="--root ${words[i]#--root=}"
                    fi
                done
                local keys
                keys=$(gosecrets __complete-keys $env_arg $root_arg 2>/dev/null)
                COMPREPLY=($(compgen -W "$keys" -- "$cur"))
            else
                COMPREPLY=($(compgen -W "--env --root" -- "$cur"))
            fi
            ;;
        init|edit|show)
            COMPREPLY=($(compgen -W "--env --root" -- "$cur"))
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash" -- "$cur"))
            ;;
    esac
}

complete -F _gosecrets gosecrets
`

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

func resolveRoot(args *[]string) string {
	for i, arg := range *args {
		if arg == "--root" && i+1 < len(*args) {
			root := (*args)[i+1]
			*args = append((*args)[:i], (*args)[i+2:]...)

			return root
		}

		if strings.HasPrefix(arg, "--root=") {
			root := strings.TrimPrefix(arg, "--root=")
			*args = append((*args)[:i], (*args)[i+1:]...)

			return root
		}
	}

	if root := os.Getenv(store.EnvRoot); root != "" {
		return root
	}

	return ""
}
