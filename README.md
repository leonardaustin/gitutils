# GoGet - A Replacement for `go get`

Do you miss the simplicity and convenience of the `go get` command for fetching Go packages? Look no further! Introducing **GoGet**, a custom command-line tool designed to bring back the joy of fetching Go packages and more.

## What is GoGet?

GoGet is a Go-based command-line tool inspired by the deprecated `go get` command. It aims to provide a similar experience while adding additional features and flexibility for managing Go packages and repositories.

## Why GoGet?

With the deprecation of the original `go get` command, you may find yourself longing for a replacement that offers the same simplicity and ease of use. GoGet is here to fill that void, providing an alternative that meets your Go package management needs.

## Features

- Fetch Go packages and their dependencies effortlessly.
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

For example, cloning any Git repository:

```shell
goget github.com/orgname/repo1
```

GoGet will clone the repository `github.com/orgname/repo1` and save it to `~/src/github.com/orgname/repo1`.

## Disclaimer

GoGet is not affiliated with or endorsed by the official Go project or the go get command. It's a fun and independent project developed to enhance repository cloning experiences.

---

Don't let the deprecation of go get stop you from effortlessly managing your Go packages and repositories. Embrace the power of GoGet and enjoy a seamless package management experience!

Happy GoGetting!
