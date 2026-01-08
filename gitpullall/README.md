# gitpullall

Concurrently fetch and pull all Git repositories in a directory. Perfect for keeping a collection of repos up to date.

## Installation

```shell
go install github.com/leonardaustin/goget/gitpullall@latest
```

## Usage

```shell
# Update all repos in current directory
gitpullall

# Update repos in a specific directory
gitpullall -dir ~/src/github.com/myorg

# Increase parallelism for faster updates
gitpullall -workers 16

# Preview what would happen
gitpullall -dry-run

# Show detailed progress
gitpullall -verbose

# Longer timeout for large repos
gitpullall -timeout 10m
```

## Options

| Flag       | Default | Description                               |
| ---------- | ------- | ----------------------------------------- |
| `-dir`     | `.`     | Directory containing Git repositories     |
| `-workers` | `8`     | Number of concurrent git operations       |
| `-timeout` | `5m`    | Timeout per repository                    |
| `-dry-run` | `false` | Show what would be done without executing |
| `-verbose` | `false` | Show detailed output                      |

## Features

- **Parallel processing** — configurable worker pool for fast batch updates
- **Smart pulling** — tries fast-forward first, falls back to rebase
- **Safe with uncommitted changes** — fetches only, won't disrupt your work
- **Prunes stale branches** — removes deleted remote-tracking branches
- **Graceful shutdown** — Ctrl+C stops cleanly without leaving repos in bad states
- **Clear status indicators** — emoji-based output shows success ✅, warnings ⚠️, and failures ❌
- **Summary report** — overview of results with details on any issues

## How It Works

1. Scans the target directory for immediate subdirectories containing `.git`
2. For each repository:
   - Fetches from all remotes with pruning
   - Checks if behind upstream
   - If clean working tree: pulls changes (fast-forward or rebase)
   - If uncommitted changes: reports status without pulling
3. Prints a summary with counts and any failures

## Example Output

```
📂 Found 12 git repositories in /Users/me/src/github.com/myorg
🔧 Using 8 workers with 5m0s timeout per repo

✅ api-server: pulled 3 commits (main)
✅ frontend: already up to date (main)
⚠️  cli-tools: fetched only - 2 commits behind (uncommitted changes)
✅ docs: pulled 1 commits (main)
❌ old-service: fetch failed - repository not found

──────────────────────────────────────────────────
📊 Summary
──────────────────────────────────────────────────
✅ Successful: 10
⚠️  Warnings:   1
❌ Failed:     1
📁 Total:      12
```

## Requirements

- Go 1.16+ (for installation)
- Git
