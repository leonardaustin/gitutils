// Package main provides a command-line tool for cloning Git repositories
// into a structured directory layout based on the repository URL.
// Repositories are stored in ~/src/{domain}/{org}/{repo}.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var gitProgressPattern = regexp.MustCompile(`^(?:remote:\s+)?([^:]+):\s+(\d+)%`)

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
	cmd := exec.Command("git", "clone", "--progress", cloneURL, destPath)
	cmd.Stdout = os.Stdout
	progressWriter := newGitProgressWriter(os.Stderr)
	defer progressWriter.Finish()
	cmd.Stderr = progressWriter

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

func parseGitProgress(line string) (stage, percent string, ok bool) {
	matches := gitProgressPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 3 {
		return "", "", false
	}

	return matches[1], matches[2], true
}

type gitProgressWriter struct {
	out         io.Writer
	buffer      bytes.Buffer
	active      bool
	lastStage   string
	lastPercent string
}

func newGitProgressWriter(out io.Writer) *gitProgressWriter {
	return &gitProgressWriter{out: out}
}

func (w *gitProgressWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		switch b {
		case '\r', '\n':
			if err := w.flushLine(); err != nil {
				return 0, err
			}
		default:
			if err := w.buffer.WriteByte(b); err != nil {
				return 0, err
			}
		}
	}

	return len(p), nil
}

func (w *gitProgressWriter) Finish() error {
	if err := w.flushLine(); err != nil {
		return err
	}

	if w.active {
		w.resetProgress()
		_, err := fmt.Fprintln(w.out)
		return err
	}

	return nil
}

func (w *gitProgressWriter) flushLine() error {
	if w.buffer.Len() == 0 {
		return nil
	}

	rawLine := w.buffer.String()
	w.buffer.Reset()

	trimmedLine := strings.TrimSpace(rawLine)
	if trimmedLine == "" {
		return nil
	}

	if stage, percent, ok := parseGitProgress(trimmedLine); ok {
		return w.writeProgress(stage, percent)
	}

	if w.active {
		w.resetProgress()
		if _, err := fmt.Fprintln(w.out); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w.out, strings.TrimRight(rawLine, " \t"))
	return err
}

func (w *gitProgressWriter) writeProgress(stage, percent string) error {
	if w.active && w.lastStage != stage {
		if _, err := fmt.Fprintln(w.out); err != nil {
			return err
		}
	}

	if w.lastStage == stage && w.lastPercent == percent {
		w.active = true
		return nil
	}

	w.active = true
	w.lastStage = stage
	w.lastPercent = percent

	_, err := fmt.Fprintf(w.out, "\r%s: %s%%", stage, percent)
	return err
}

func (w *gitProgressWriter) resetProgress() {
	w.active = false
	w.lastStage = ""
	w.lastPercent = ""
}
