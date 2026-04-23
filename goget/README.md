# GoGet

A modern replacement for the classic `go get` behavior that clones any Git repository into a clean, organized directory structure.

## Why GoGet?

The new `go get` command only works with Go modules and can't fetch non-Go repositories. GoGet brings back the simplicity of the old approach—clone any Git repo and store it in a predictable location: `~/src/{domain}/{org}/{repo}`.

## Installation

```shell
go install github.com/leonardaustin/goget/goget@latest
```

## Usage

```shell
goget <repo-url>
```

### Examples

```shell
# Standard domain/org/repo format
goget github.com/golang/go
# → Clones to ~/src/github.com/golang/go

# HTTPS URLs
goget https://github.com/kubernetes/kubernetes
# → Clones to ~/src/github.com/kubernetes/kubernetes

# SSH URLs
goget git@github.com:docker/compose.git
# → Clones to ~/src/github.com/docker/compose

# Git protocol
goget git://github.com/torvalds/linux
# → Clones to ~/src/github.com/torvalds/linux
```

## Features

- **Any Git repository** — works with Go and non-Go projects alike
- **Multiple URL formats** — handles HTTPS, HTTP, SSH (`git@`), and `git://` protocols
- **Clean paths** — automatically strips `.git` suffix and normalizes URLs
- **Safe operation** — won't overwrite existing repositories
- **Clone progress** — shows live percentage updates while Git downloads objects
- **Helpful errors** — provides clear messages for common failures (auth issues, network problems, invalid URLs)

## How It Works

1. Normalizes the input URL (strips protocol prefixes, converts SSH format, removes `.git` suffix)
2. Extracts domain, organization, and repository name
3. Creates the directory structure under `~/src/`
4. Clones the repository using HTTPS

## Requirements

- Go 1.16+ (for installation)
- Git

---

*Not affiliated with the official Go project.*
