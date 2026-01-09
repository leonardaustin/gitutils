# clone-all

Clone all repositories from a GitHub organization with concurrent workers, progress tracking, and configuration persistence.

## Prerequisites

- [GitHub CLI (gh)](https://cli.github.com/) - must be authenticated (`gh auth login`)
- Git

## Installation

```bash
cd gitcloneall
go build -o clone-all .
mv clone-all /usr/local/bin/  # or ~/bin/
```

## Quick Start

```bash
# Set your default organization
clone-all init -org mycompany

# Clone all repos
clone-all clone

# Clone to a specific directory
clone-all clone -dir ~/projects/mycompany
```

## Commands

### `clone-all init`

Initialize configuration (saved to `~/.clone-all-config.json`).

| Flag       | Default  | Description                        |
| ---------- | -------- | ---------------------------------- |
| `-org`     | required | Default organization to clone from |
| `-branch`  | `master` | Default branch to checkout         |
| `-workers` | `8`      | Number of concurrent workers       |

### `clone-all clone`

Clone repositories from an organization.

| Flag       | Default       | Description                    |
| ---------- | ------------- | ------------------------------ |
| `-org`     | (from config) | Organization to clone from     |
| `-branch`  | (from config) | Branch to checkout             |
| `-workers` | (from config) | Number of concurrent workers   |
| `-limit`   | `1000`        | Maximum repos to fetch         |
| `-dir`     | `.`           | Directory to clone into        |
| `-dry-run` | `false`       | Preview without cloning        |
| `-force`   | `false`       | Re-clone existing repos        |
| `-ssh`     | `true`        | Use SSH (false for HTTPS)      |
| `-filter`  |               | Filter repos by name substring |

### `clone-all config -show`

Display current configuration.

## Examples

```bash
# Preview what would be cloned
clone-all clone -dry-run

# Clone only repos containing "api" in the name
clone-all clone -filter api

# Use HTTPS instead of SSH with more workers
clone-all clone -ssh=false -workers 16

# Force re-clone existing repos
clone-all clone -force
```

## Output

```
$ clone-all clone -dir ~/work/myorg
🔄 Fetching repository list from myorg...
📂 Found 47 repositories in myorg
🔧 Using 8 workers

✅ api-service: cloned (3s)
✅ web-app: cloned (5s)
⚠️  legacy-repo: skipped (already exists)
❌ archived-thing: clone failed: repository not accessible

──────────────────────────────────────────────────
📊 Summary
──────────────────────────────────────────────────
✅ Cloned:  32
⚠️  Skipped: 14
❌ Failed:  1
📁 Total:   47
```

## License

MIT
