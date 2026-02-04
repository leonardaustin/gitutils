# Go Git Utilities

A collection of command-line tools for managing Git repositories.

## Tools

### [goget](./goget/)

Clone any Git repository into `~/src/{domain}/{org}/{repo}`. A modern replacement for the classic `go get` cloning behavior.

```shell
go install github.com/leonardaustin/goget/goget@latest
goget github.com/golang/go
```

### [gitpullall](./gitpullall/)

Concurrently fetch and pull all Git repositories in a directory.

```shell
go install github.com/leonardaustin/goget/gitpullall@latest
gitpullall -dir ~/src/github.com
```

### [gitcloneall](./gitcloneall/)

Clone all repositories from a GitHub organization concurrently. Supports configuration persistence, dry runs, filtering, and SSH/HTTPS cloning.

```shell
go install github.com/leonardaustin/goget/gitcloneall@latest
gitcloneall init -org mycompany
gitcloneall clone
```

Requires [GitHub CLI (`gh`)](https://cli.github.com/) to be installed and authenticated.
