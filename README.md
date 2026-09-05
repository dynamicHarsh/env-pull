# inject

**Inject a project's secrets into a foreground child process, without a plaintext `.env` file.**

`inject` is a command-line adapter, not a secret manager. It reads a non-secret `inject.toml` configuration, retrieves a complete secret set from your existing provider or the OS credential store, and supplies it only to the child process it starts.

The legacy `env-pull` binary remains a temporary compatibility alias and prints a migration notice.

## Why

Plaintext `.env` files persist on developer machines, can be committed accidentally, and tend to drift from the approved secret source. `inject` keeps secret values off the project disk while preserving normal development commands through named bindings.

- Secrets are passed only through process environment inheritance.
- Project configuration is safe to commit: it contains provider references, never values or provider credentials.
- Remote providers remain the system of record for authentication, authorization, audit history, and rotation.
- A child process receives a complete validated secret set or does not start.

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap dynamicHarsh/tap && brew install inject
```

### Scoop (Windows)

```powershell
scoop bucket add env-pull https://github.com/dynamicHarsh/scoop-bucket
scoop install inject
```

### Build from source

```bash
git clone https://github.com/harsh-sonkar/env-pull.git
cd env-pull
make build
```

The binary is written to `./bin/inject`.

## Quick Start

### Team project: 1Password

Create a Secure Note in your team's 1Password vault. Its body must use standard `.env` syntax:

```dotenv
DATABASE_URL=postgres://...
API_KEY=...
```

Install and sign in to the 1Password CLI, then run setup from the project root. Setup previews the non-secret configuration before making changes.

```bash
op signin
inject setup \
  --project-id acme-web \
  --account acme.1password.com \
  --vault Engineering \
  --item acme-web-development \
  --binding dev \
  --command npm --command run --command dev:app \
  --yes
```

Commit the generated `inject.toml`. Each developer signs in to `op` using their own account, then runs the binding:

```bash
inject dev
```

### Local-only project

From a directory with an existing `.env`, setup imports into macOS Keychain or Linux Secret Service by default. The project ID defaults to the directory name and can be overridden with `--project-id`:

```bash
inject setup \
  --binding dev \
  --command npm --command run --command dev \
  --yes
```

The imported values stay in the operating system credential store. They are not shared through `inject.toml`, and `.env` remains unless removal is explicitly requested. Use `--local` to force local source selection when needed.

### One-off command

Run any command using the default profile without creating a binding:

```bash
inject run -- npm run db:migrate
inject run --profile staging -- ./bin/server --check
```

Injected values replace same-named variables inherited from the invoking shell. They are not exported back to that shell.

## Configuration

`inject.toml` is committed at the project root. It contains no secret values, provider tokens, private item URLs, or developer-specific paths.

```toml
format_version = 1
project_id = "acme-web"

[profiles.default]
provider = "1password"
account = "acme.1password.com"
vault = "Engineering"
item = "acme-web-development"
item_id = "provider-item-id"

[commands]
dev = { command = ["npm", "run", "dev:app"] }

[cache]
enabled = false
max_age = "24h"
```

Supported profile providers are `1password`, `bitwarden`, and `local`. A `local` profile needs no provider reference. Provider item IDs take precedence over display names when both are configured.

For a named profile, set the binding's `profile` field:

```toml
[profiles.staging]
provider = "bitwarden"
item = "acme-web-staging"

[commands.staging]
profile = "staging"
command = ["npm", "run", "dev"]
```

## Providers and Offline Use

Remote secret notes are managed only by their provider. `inject` reuses the authenticated session from the relevant CLI:

- 1Password: `op signin`
- Bitwarden: `bw login`

Normal remote runs always fetch fresh values. You may explicitly enable a credential-store cache in `inject.toml` and request it for offline work:

```toml
[cache]
enabled = true
max_age = "24h"
```

```bash
inject run --offline -- npm run dev
```

Offline use fails when the cache is disabled, unavailable, expired, or used in CI.

## Command Reference

```text
inject setup [flags]
inject <binding>
inject run [--profile <name>] [--offline] -- <command> [args...]
inject remove --yes
inject edit                 # legacy encrypted-vault workflow
inject export               # legacy encrypted-vault export
```

- `setup` previews and, with `--yes`, creates or updates configuration and optional command bindings. `--local` imports a legacy `.env` into the credential store. Use `--validate` repeatedly to supply a finite validation command; `--remove-env --yes-remove-env` removes a detected legacy `.env` only after validation.
- `<binding>` runs a named command from `inject.toml` as an injected foreground child process.
- `run` injects a selected profile into an arbitrary child command. Place all `inject` flags before the child command.
- `remove --yes` deletes `inject.toml`, this project's local credential-store values, and remote caches. It never deletes remote provider items.
- `edit` and `export` remain for compatibility with the former encrypted `.env.pull.enc` workflow. New projects should use `setup` instead.

Run `inject <command> --help` for detailed flags; because `run` passes its arguments through unchanged, use `inject help run` for that command's help.

## Security Model

| Property | Behavior |
| --- | --- |
| Project disk | `inject.toml` contains no secret values. |
| Environment scope | Secrets are available only to the child process tree. |
| Parent shell | The invoking shell is never modified. |
| Provider access | Authentication and authorization stay with 1Password or Bitwarden. |
| Local values | Stored in macOS Keychain or Linux Secret Service. |
| Failure mode | Invalid configuration, unavailable source, or malformed secret sets prevent process launch. |
| Output | Child stdout and stderr pass through unchanged; `inject` does not redact application output. |

## Contributing

```bash
go test ./...
make build
make test
```

## License

MIT. See [LICENSE](LICENSE).
