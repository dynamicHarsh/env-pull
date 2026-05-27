# env-pull

**A Universal Secrets Adapter. Zero-disk. Zero-config.**

> Stop putting secrets in `.env` files. Start injecting them.

---

## The Problem

Every development team eventually develops the same bad habit: a plaintext `.env` file that holds production credentials, gets shared over Slack, lives on laptops as-is, and occasionally makes its way into a git commit. Even when teams use a secrets manager, the workflow usually devolves into:

1. Log into the vault UI.
2. Copy the secret.
3. Paste it into `.env`.
4. Hope nobody commits it.

The `.env` file is a liability: it is plaintext, it is persistent, and it is always one `git add .` away from being public.

---

## The Solution: Zero-Disk Injection

`env-pull` is a thin process wrapper. It fetches secrets from your vault at runtime and injects them directly into your target command's environment via OS-level process tree inheritance. The secrets:

- **Never touch the filesystem** as plaintext.
- **Never appear in logs** or shell history.
- **Vanish automatically** the moment your process exits — no cleanup required.

<img width="791" height="701" alt="Screenshot 2026-05-27 212844" src="https://github.com/user-attachments/assets/b0e6e0aa-5f78-4b8c-9c4a-d8d31f319add" />

---

## Installation

### Homebrew (macOS / Linux) — coming soon
```bash
brew install harsh-sonkar/tap/env-pull
```

### GitHub Releases
Download a pre-built binary for your platform from the [Releases](https://github.com/harsh-sonkar/env-pull/releases) page:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/harsh-sonkar/env-pull/releases/latest/download/env-pull_latest_darwin_arm64.tar.gz | tar xz
sudo mv env-pull /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/harsh-sonkar/env-pull/releases/latest/download/env-pull_latest_linux_amd64.tar.gz | tar xz
sudo mv env-pull /usr/local/bin/
```

### Build from source
```bash
git clone https://github.com/harsh-sonkar/env-pull.git
cd env-pull
go build -o env-pull .
```

---

## Quick Start

### Option A — Local Encrypted Vault (`env-pull edit`)

Use this when you want a simple, offline-capable, team-shareable workflow where secrets are stored encrypted in your repo or home directory.

**Step 1: Add your secrets**
```bash
env-pull edit
```
Your `$EDITOR` opens with a standard `.env`-format file:
```dotenv
# .env.pull.enc (decrypted view — never written to disk as-is)
DB_PASSWORD=s3cr3t
API_KEY=abc123
STRIPE_SECRET_KEY=sk_live_...
```
Save and exit. The file is re-encrypted with AES-256-GCM and the plaintext is wiped.

**Step 2: Inject at runtime**
```bash
# Wrap any command — it sees the secrets as environment variables
env-pull run -- ./my-server
env-pull run -- npm start
env-pull run -- python manage.py runserver
env-pull run -- psql --host=localhost mydb
```

---

### Option B — AWS Secrets Manager (`--aws-secret`)

Use this for team or CI environments where secrets are managed centrally. No new credentials are needed — `env-pull` reuses your existing AWS auth context.

**Prerequisites:**
- AWS credentials configured via any standard method:
  - `~/.aws/credentials` (shared credentials file)
  - `AWS_PROFILE` environment variable
  - `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` environment variables
  - EC2 / ECS / EKS IAM instance role

**Store your secret in AWS Secrets Manager as a JSON object:**
```json
{
  "DB_PASSWORD": "s3cr3t",
  "API_KEY":     "abc123"
}
```

**Inject at runtime:**
```bash
env-pull run --aws-secret prod/my-app -- ./my-server
env-pull run --aws-secret prod/my-app -- npm start

# The child process sees DB_PASSWORD and API_KEY as normal env vars
env-pull run --aws-secret prod/my-app -- printenv DB_PASSWORD
```

---

## How It Works

```
env-pull run --aws-secret prod/db -- psql mydb
      │
      ├─ 1. Load AWS default credential chain (zero new config)
      ├─ 2. Call secretsmanager:GetSecretValue("prod/db")
      ├─ 3. Parse JSON payload → map[string]string in memory
      ├─ 4. Merge with current environment (no overwrites of existing vars)
      ├─ 5. exec(psql, env=[...existing..., DB_PASSWORD=s3cr3t, ...])
      └─ 6. psql exits → memory is reclaimed, secrets are gone
```

The local vault path is identical, but step 1–3 are replaced by:
```
      ├─ 1. Read .env.pull.enc
      ├─ 2. Decrypt with AES-256-GCM using ~/.config/env-pull/master.key
      ├─ 3. Parse .env format → map[string]string in memory
```

---

## Key Management

The local vault is encrypted with a 32-byte AES-256-GCM key stored at:

```
~/.config/env-pull/master.key   (permissions: 0600)
~/.config/env-pull/             (permissions: 0700)
```

The key is generated automatically on the first `env-pull edit`. **Back it up.**
Without it, the vault cannot be decrypted.

---

## Command Reference

```
env-pull [command]

Commands:
  edit    Open the encrypted secrets vault in your default editor
  run     Run a command with vault secrets injected into its environment

Flags:
  -h, --help   Show help

Run 'env-pull <command> --help' for detailed usage.
```

---

## Security Model

| Property              | Guarantee                                                  |
|-----------------------|------------------------------------------------------------|
| Disk exposure         | Plaintext secrets never written to disk                    |
| Log exposure          | Secret values never passed to any logger or stdout         |
| Memory lifetime       | Byte buffers are zeroed immediately after use              |
| Temp file wipe        | Editor temp file overwritten with zeros before deletion    |
| Encryption            | AES-256-GCM with per-message random nonce                  |
| Key permissions       | Master key stored at 0600; directory at 0700               |

> **Note on physical-layer guarantees:** Zeroing buffers and temp files is
> best-effort at the software layer. On copy-on-write or log-structured
> filesystems (APFS, ZFS, most NVMe SSDs), the OS may retain previous data
> at the hardware level. `env-pull` mitigates risk at the software layer; it
> does not make guarantees about physical storage media.

---

## 🚀 Enterprise Control Plane — Coming Soon

A centralised audit and governance layer is in active development. It will include:

- **Audit logs** — every secret fetch recorded with timestamp, user, and command.
- **Policy engine** — define which identities can access which secrets in which environments.
- **Secret rotation hooks** — automatically re-inject rotated credentials without restarting processes.
- **Team vaults** — shared encrypted vaults with per-member access control.
- **SSO / OIDC integration** — replace static AWS credentials with short-lived OIDC tokens.

[Sign up for early access →](https://github.com/harsh-sonkar/env-pull)

---

## Contributing

```bash
git clone https://github.com/harsh-sonkar/env-pull.git
cd env-pull
go test ./...        # run all tests
make build           # compile to ./bin/env-pull
make test            # run tests
```

---

## License

MIT — see [LICENSE](LICENSE).
