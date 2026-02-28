![Version](https://img.shields.io/badge/version-0.0.0-orange.svg)
![Go](https://img.shields.io/github/go-mod/go-version/bilustek/gosecrets)
[![Documentation](https://godoc.org/github.com/bilustek/gosecrets?status.svg)](https://pkg.go.dev/github.com/bilustek/gosecrets)
[![Go Report Card](https://goreportcard.com/badge/github.com/bilustek/gosecrets)](https://goreportcard.com/report/github.com/bilustek/gosecrets)
[![codecov](https://codecov.io/gh/bilustek/gosecrets/graph/badge.svg?token=7EADRS8WY2)](https://codecov.io/gh/bilustek/gosecrets)
[![Run go tests](https://github.com/bilustek/gosecrets/actions/workflows/go-test.yml/badge.svg)](https://github.com/bilustek/gosecrets/actions/workflows/go-test.yml)
[![Run golangci-lint](https://github.com/bilustek/gosecrets/actions/workflows/go-lint.yml/badge.svg)](https://github.com/bilustek/gosecrets/actions/workflows/go-lint.yml)

# gosecrets

Encrypted credentials management for Go projects - inspired by [Rails credentials][001].

Stop juggling `.env` files and environment variables. Keep your secrets
encrypted **in your repo**, share them safely **with your team**, and access them with
a clean API.

---

## Example Flow

First, create stores for development and production environments:

```bash
gosecrets init                    # secrets/master.key + secrets/credentials.enc
gosecrets init --env production   # secrets/production.key + secrets/production.enc
```

Edit your **development** config: `gosecrets edit` (pops up your default `$EDITOR`), edit `yaml`:

```yaml
stripe:
  key: sk_test_xxx
  webhook_secret: whsec_test_abc
```

Edit your **production** config: `gosecrets edit --env production`:

```yaml
stripe:
  key: sk_live_yyy
  webhook_secret: whsec_live_def
```

Fix your `.gitignore` file

    secrets/*.key

Keep this in mind! **NEVER COMMIT** `*.key` files!

Now, in your go project;

```go
secrets, err := gosecrets.Load() // for development variables
```

For production;

```go
secrets, err := gosecrets.Load(gosecrets.WithEnv("production"))
if err != nil {
    // handle your error
}

fmt.Println(secrets.String("stripe.key"))
// development: sk_test_xxx
// production:  sk_live_yyy
```

You can choose any environment name, such as `foo`;

```bash
export GOSECRETS_FOO_KEY="my-super-secret-foo-key"
gosecrets init --env foo
```

Then in your go project;

```go
secrets, err := gosecrets.Load(gosecrets.WithEnv("foo"))
```

| Environment | .enc file | key from? | ENV VAR |
|:------------|:-----|:-----|:-----|
| development | `secrets/credentials.enc` | `secrets/master.key` (disc) | not required |
| production  | `secrets/production.enc` | env vars | `GOSECRETS_PRODUCTION_KEY` |
| <any-name>  | `secrets/<any-name>.enc` | env vars | `GOSECRETS_<ANY-NAME>_KEY` |


---

## How It Works ?

`gosecrets` stores your secrets **encrypted inside your repository**. Only 
**the master key** stays **outside version control**.

After running `gosecrets init`, your project gets:

    your-project/
    ├── secrets/
    │   ├── credentials.enc   # encrypted YAML — committed to git
    │   └── master.key        # decryption key — add to .gitignore!

With `--env` flag, each environment gets its own pair:

    your-project/
    ├── secrets/
    │   ├── credentials.enc
    │   ├── master.key
    │   ├── production.enc
    │   ├── production.key
    │   ├── staging.enc
    │   └── staging.key

The master key is resolved in this order:

1. `GOSECRETS_MASTER_KEY` environment variable
1. `GOSECRETS_<ENV>_KEY` environment variable (e.g. `GOSECRETS_PRODUCTION_KEY`)
1. Key file on disk (`secrets/master.key` or `secrets/<env>.key`)

In **development**, the key file on disk is enough. In **production/CI**, set
the environment variable and never deploy the key file.

---

## Encryption

Credentials are encrypted with **AES-256-GCM** (*authenticated encryption*).
Each write generates a fresh random nonce - the same plaintext produces
different **ciphertext** every time.

    ┌──────────┐     ┌──────────────┐     ┌─────────────────┐
    │ You edit │────▶│ gosecrets    │────▶│ credentials.enc │
    │ YAML     │     │ encrypts     │     │ (committed)     │
    └──────────┘     └──────────────┘     └─────────────────┘
    
    ┌──────────┐     ┌──────────────┐     ┌─────────────────┐
    │ Your app │◀────│ gosecrets    │◀────│ credentials.enc │
    │ reads    │     │ decrypts     │     │ + master key    │
    └──────────┘      └──────────────┘     └─────────────────┘

---

## Installation

Library:

```bash
go get -u github.com/bilustek/gosecrets
```

CLI Tool:

```bash
go install github.com/bilustek/gosecrets@latest
```

---

## Usage

@wip

---

## Contributor(s)

- [Uğur "vigo" Özyılmazel](https://github.com/vigo) - Creator, maintainer

---

## Contribute

All PR’s are welcome!

1. `fork` (https://github.com/bilustek/gosecrets/fork)
1. Create your `branch` (`git checkout -b my-feature`)
1. `commit` yours (`git commit -am 'add some functionality'`)
1. `push` your `branch` (`git push origin my-feature`)
1. Than create a new **Pull Request**!

---

## License

This project is licensed under MIT

---

This project is intended to be a safe, welcoming space for collaboration, and
contributors are expected to adhere to the [code of conduct][coc].

[coc]: https://github.com/bilustek/gosecrets/blob/main/CODE_OF_CONDUCT.md
[001]: https://edgeguides.rubyonrails.org/security.html#custom-credentials
