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
