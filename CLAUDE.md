# Project: env-pull
## Overview
`env-pull` is a Universal Secrets Adapter CLI. It securely fetches credentials from upstream enterprise vaults (e.g., AWS, 1Password) or an encrypted local file (`env-edit`) and injects them directly into an application's memory via process tree inheritance. 

## Tech Stack
- **Language:** Go (Golang) 1.21+
- **CLI Framework:** Cobra
- **Execution:** Native OS process spawning / Sub-shell injection
- **Target:** Statically linked binaries for macOS, Linux, and Windows

## Core Architectural Rules
1. **Zero-Disk (Absolute Rule):** NEVER write plaintext secrets to the filesystem. 
2. **Sub-shell Injection:** Wrap the developer's process. Fetch secrets, spawn a new child process/sub-shell with the secrets in memory, and ensure they vanish when the process terminates.
3. **Zero-Config First:** Rely on existing developer authentication contexts (e.g., `~/.aws/credentials`, 1Password IPC) rather than requiring new tokens.
4. **Performance:** The CLI must boot and execute in under 50ms to maintain seamless developer flow.

## Project Structure (Target)
- `/cmd`: Cobra command entry points.
- `/internal`: Core business logic (Process injection, crypto, vault adapters).
- `/pkg`: Exportable utilities (if any).