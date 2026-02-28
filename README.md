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
gosecrets init                    # secrets/development.key + secrets/development.enc
gosecrets init --env production   # secrets/production.key + secrets/production.enc
```

Each `init` prints a master key — **save it somewhere safe**. You will need
this key to decrypt your credentials in deployment (e.g. Portainer, CI).
Locally, the key file on disk is used automatically.

You should see something like:

    master key: e5d0529e.................................................
    save this key somewhere safe, you need it to decrypt your credentials.

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

Now, in your Go project, your code is **the same everywhere**:

```go
secrets, err := gosecrets.Load()
if err != nil {
    // handle your error
}

fmt.Println(secrets.String("stripe.key"))
// development: sk_test_xxx
// production:  sk_live_yyy
```

`gosecrets.Load()` picks the environment automatically from `GOSECRETS_ENV`.
Default is `development`. In production, just set:

```bash
export GOSECRETS_ENV=production
export GOSECRETS_PRODUCTION_KEY="your-master-key-here"

# or whatever your env is:
# export GOSECRETS_STAGING_KEY="your-master-key-here"
# export GOSECRETS_FOO_KEY="your-master-key-here"
```

You can choose any environment name, such as `foo`:

```bash
export GOSECRETS_ENV=foo
gosecrets init                     # creates secrets/foo.key + secrets/foo.enc
gosecrets edit                     # add your secrets
```

```go
secrets, err := gosecrets.Load()   // picks up GOSECRETS_ENV=foo automatically
```

| Environment | `GOSECRETS_ENV` | .enc file | Key from | Key ENV VAR |
|:------------|:-----|:-----|:-----|:-----|
| development | _(empty or unset)_ | `secrets/development.enc` | `secrets/development.key` (disk) | `GOSECRETS_DEVELOPMENT_KEY` |
| production  | `production` | `secrets/production.enc` | env var | `GOSECRETS_PRODUCTION_KEY` |
| \<any-name\>  | `<any-name>` | `secrets/<any-name>.enc` | env var | `GOSECRETS_<ANY-NAME>_KEY` |


---

## How It Works ?

`gosecrets` stores your secrets **encrypted inside your repository**. Only
**the master key** stays **outside version control**.

Each environment gets its own `.enc` / `.key` pair inside `secrets/`:

    your-project/
    ├── secrets/
    │   ├── development.enc   # encrypted YAML — committed to git
    │   ├── development.key   # decryption key — add to .gitignore!
    │   ├── production.enc
    │   └── production.key

In **development**, the key file on disk is enough. In **production/CI**, set
`GOSECRETS_<ENV>_KEY` (e.g. `GOSECRETS_PRODUCTION_KEY`) as an environment
variable and never deploy the key file.

> `.key` files are for local work, env vars are for deployment.
> When both exist, the env var takes precedence.

---

## Encryption

Credentials are encrypted with **AES-256-GCM** (*authenticated encryption*).
Each write generates a fresh random nonce - the same plaintext produces
different **ciphertext** every time.

    ┌──────────┐     ┌──────────────┐     ┌─────────────────┐
    │ You edit │────▶│ gosecrets    │────▶│ <env>.enc       │
    │ YAML     │     │ encrypts     │     │ (committed)     │
    └──────────┘     └──────────────┘     └─────────────────┘
    
    ┌──────────┐     ┌──────────────┐     ┌─────────────────┐
    │ Your app │◀────│ gosecrets    │◀────│ <env>.enc       │
    │ reads    │     │ decrypts     │     │ + master key    │
    └──────────┘     └──────────────┘     └─────────────────┘

---

## Installation

Library:

```bash
go get -u github.com/bilustek/gosecrets
```

CLI Tool:

```bash
go install github.com/bilustek/gosecrets/cmd/gosecrets@latest
```

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
