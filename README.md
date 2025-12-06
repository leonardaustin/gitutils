# GoGet - A Replacement for `go get`

Do you miss the simplicity and convenience of the old school `go get` command for fetching any git repo and putting them in the right place? Look no further! Introducing **GoGet**, a custom command-line tool designed to bring back the joy of fetching Go packages and more.

## What is GoGet?

GoGet is a Go-based command-line tool inspired by previous version of the `go get` command. It aims to provide a similar experience while adding additional features and flexibility for managing Go packages and repositories.

## Why GoGet?

With new `go get` command you are unable to get non golang sources and so you may find yourself longing for a replacement that offers the same simplicity and ease of use. GoGet is here to fill that void, providing an alternative that meets your Go package management needs.

## Features

- Fetch git sourced repos.
- Clone Git repositories and store them in predefined paths.
- Support for Go modules and non-Go repositories alike.
- Flexibility to specify custom storage locations for repositories.

## Usage

To use GoGet, simply follow these steps:

1. Ensure you have Go and Git installed on your machine.

2. Install the GoGet command-line tool.

```shell
go install github.com/leonardaustin/goget@latest
goget github.com/orgname/repo1
```

## How GoGet Works

GoGet extracts the base domain from the repository URL you provide and clones the repository into the desired path. It dynamically determines the correct organization and repository names to create a clean directory structure.

For exmaple, cloning any Git repository
To clone a Git repository and store it in a predefined path, use the GoGet command followed by the repository URL:

```shell
goget github.com/orgname/repo1
```

GoGet will clone the repository github.com/orgname/repo1 and save it to the path ~/src/github.com/orgname.

## Disclaimer

GoGet is not affiliated with or endorsed by the official Go project or the go get command.

---

Happy GoGetting!
