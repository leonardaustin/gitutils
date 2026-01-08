// Package main provides a command-line tool for cloning Git repositories
// into a structured directory layout based on the repository URL.
// Repositories are stored in ~/src/{domain}/{org}/{repo}.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: goget <repo-url>")
		os.Exit(1)
	}

	repoURL := os.Args[1]
	normalizedURL := normalizeURL(repoURL)
	destPath := getDestPath(normalizedURL)

	// Check if destination already exists to avoid overwriting
	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("Repository already exists at %s\n", destPath)
		os.Exit(0)
	}

	if err := gitClone(normalizedURL, destPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error cloning repository: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Repository cloned to %s\n", destPath)
}

// normalizeURL converts various Git URL formats into a consistent
// domain/org/repo format. It handles:
//   - HTTPS URLs: https://github.com/org/repo
//   - HTTP URLs: http://github.com/org/repo
//   - Git protocol: git://github.com/org/repo
//   - SSH URLs: git@github.com:org/repo.git
//   - URLs with .git suffix
func normalizeURL(repoURL string) string {
	// Strip common protocol prefixes
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "http://")
	repoURL = strings.TrimPrefix(repoURL, "git://")

	// Convert SSH format (git@host:path) to standard path format (host/path)
	if strings.HasPrefix(repoURL, "git@") {
		repoURL = strings.TrimPrefix(repoURL, "git@")
		repoURL = strings.Replace(repoURL, ":", "/", 1)
	}

	// Remove .git suffix for cleaner directory names
	repoURL = strings.TrimSuffix(repoURL, ".git")
	return repoURL
}

// getDestPath constructs the local filesystem path where the repository
// will be cloned. The path follows the pattern: ~/src/{domain}/{org}/{repo}
// For example: github.com/golang/go -> ~/src/github.com/golang/go
func getDestPath(repoURL string) string {
	parts := strings.Split(repoURL, "/")

	// Validate URL has at least domain, org, and repo components
	if len(parts) < 3 {
		fmt.Fprintln(os.Stderr, "Invalid repository URL: expected format domain/org/repo")
		os.Exit(1)
	}

	domain := parts[0]
	organization := parts[1]
	repo := parts[2]

	return filepath.Join(os.Getenv("HOME"), "src", domain, organization, repo)
}

// gitClone executes git clone to fetch the repository from the remote URL
// and store it at the specified destination path. It creates the parent
// directories if they don't exist.
func gitClone(repoURL, destPath string) error {
	// Create parent directory structure (e.g., ~/src/github.com/org/)
	// Note: We create the parent, not destPath itself, because git clone
	// expects to create the final directory
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Execute git clone with HTTPS protocol
	cloneURL := "https://" + repoURL
	cmd := exec.Command("git", "clone", cloneURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return formatGitError(err, cloneURL)
	}
	return nil
}

// formatGitError converts git exit codes into user-friendly error messages.
// Git uses specific exit codes to indicate different failure conditions.
func formatGitError(err error, cloneURL string) error {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return err
	}

	exitCode := exitErr.ExitCode()

	switch exitCode {
	case 1:
		// General error - could be various issues
		return fmt.Errorf("git clone failed: general error (check repository URL and permissions)")
	case 128:
		// Fatal error - most common, includes several scenarios
		return fmt.Errorf("git clone failed: %s\n"+
			"  Possible causes:\n"+
			"  - Repository does not exist or is private\n"+
			"  - Network connection issue\n"+
			"  - Invalid repository URL\n"+
			"  - Authentication required (try: git config --global credential.helper store)",
			cloneURL)
	case 129:
		// Invalid options passed to git
		return fmt.Errorf("git clone failed: invalid git options")
	default:
		return fmt.Errorf("git clone failed with exit code %d", exitCode)
	}
}
