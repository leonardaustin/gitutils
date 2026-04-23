package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain URL without protocol",
			input:    "github.com/golang/go",
			expected: "github.com/golang/go",
		},
		{
			name:     "HTTPS URL",
			input:    "https://github.com/golang/go",
			expected: "github.com/golang/go",
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/golang/go",
			expected: "github.com/golang/go",
		},
		{
			name:     "git protocol URL",
			input:    "git://github.com/golang/go",
			expected: "github.com/golang/go",
		},
		{
			name:     "SSH URL",
			input:    "git@github.com:golang/go",
			expected: "github.com/golang/go",
		},
		{
			name:     "HTTPS URL with .git suffix",
			input:    "https://github.com/golang/go.git",
			expected: "github.com/golang/go",
		},
		{
			name:     "SSH URL with .git suffix",
			input:    "git@github.com:golang/go.git",
			expected: "github.com/golang/go",
		},
		{
			name:     "GitLab URL",
			input:    "https://gitlab.com/inkscape/inkscape",
			expected: "gitlab.com/inkscape/inkscape",
		},
		{
			name:     "Bitbucket SSH URL",
			input:    "git@bitbucket.org:atlassian/python-bitbucket.git",
			expected: "bitbucket.org/atlassian/python-bitbucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDestPath(t *testing.T) {
	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set a predictable HOME for testing
	testHome := "/home/testuser"
	os.Setenv("HOME", testHome)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub repository",
			input:    "github.com/golang/go",
			expected: filepath.Join(testHome, "src", "github.com", "golang", "go"),
		},
		{
			name:     "GitLab repository",
			input:    "gitlab.com/inkscape/inkscape",
			expected: filepath.Join(testHome, "src", "gitlab.com", "inkscape", "inkscape"),
		},
		{
			name:     "custom domain repository",
			input:    "git.example.com/myorg/myrepo",
			expected: filepath.Join(testHome, "src", "git.example.com", "myorg", "myrepo"),
		},
		{
			name:     "URL with extra path segments",
			input:    "github.com/org/repo/extra/path",
			expected: filepath.Join(testHome, "src", "github.com", "org", "repo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDestPath(tt.input)
			if result != tt.expected {
				t.Errorf("getDestPath(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURLPreservesCase(t *testing.T) {
	// Repository names can be case-sensitive on some platforms
	input := "github.com/MyOrg/MyRepo"
	expected := "github.com/MyOrg/MyRepo"

	result := normalizeURL(input)
	if result != expected {
		t.Errorf("normalizeURL should preserve case: got %q, expected %q", result, expected)
	}
}

func TestNormalizeURLHandlesTrailingSlash(t *testing.T) {
	// Some URLs might have a trailing slash
	input := "https://github.com/golang/go/"
	// Note: current implementation doesn't strip trailing slash
	// This test documents the current behavior
	result := normalizeURL(input)
	expected := "github.com/golang/go/"

	if result != expected {
		t.Errorf("normalizeURL(%q) = %q, expected %q", input, result, expected)
	}
}

func TestIntegration_NormalizeAndGetDestPath(t *testing.T) {
	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := "/home/testuser"
	os.Setenv("HOME", testHome)

	tests := []struct {
		name         string
		inputURL     string
		expectedPath string
	}{
		{
			name:         "full HTTPS URL to destination path",
			inputURL:     "https://github.com/golang/go",
			expectedPath: filepath.Join(testHome, "src", "github.com", "golang", "go"),
		},
		{
			name:         "SSH URL to destination path",
			inputURL:     "git@github.com:kubernetes/kubernetes.git",
			expectedPath: filepath.Join(testHome, "src", "github.com", "kubernetes", "kubernetes"),
		},
		{
			name:         "plain URL to destination path",
			inputURL:     "github.com/docker/compose",
			expectedPath: filepath.Join(testHome, "src", "github.com", "docker", "compose"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeURL(tt.inputURL)
			destPath := getDestPath(normalized)

			if destPath != tt.expectedPath {
				t.Errorf("Integration test failed for %q:\n  normalized: %q\n  destPath: %q\n  expected: %q",
					tt.inputURL, normalized, destPath, tt.expectedPath)
			}
		})
	}
}

func TestFormatGitError(t *testing.T) {
	testURL := "https://github.com/nonexistent/repo"

	tests := []struct {
		name            string
		exitCode        int
		expectedContain string
	}{
		{
			name:            "exit code 1 - general error",
			exitCode:        1,
			expectedContain: "general error",
		},
		{
			name:            "exit code 128 - fatal error (repo not found)",
			exitCode:        128,
			expectedContain: "Repository does not exist",
		},
		{
			name:            "exit code 128 - includes URL in message",
			exitCode:        128,
			expectedContain: testURL,
		},
		{
			name:            "exit code 129 - invalid options",
			exitCode:        129,
			expectedContain: "invalid git options",
		},
		{
			name:            "unknown exit code",
			exitCode:        42,
			expectedContain: "exit code 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock ExitError by running a command that exits with specific code
			var cmd *exec.Cmd
			switch tt.exitCode {
			case 1:
				cmd = exec.Command("sh", "-c", "exit 1")
			case 128:
				cmd = exec.Command("sh", "-c", "exit 128")
			case 129:
				cmd = exec.Command("sh", "-c", "exit 129")
			default:
				cmd = exec.Command("sh", "-c", fmt.Sprintf("exit %d", tt.exitCode))
			}

			err := cmd.Run()
			if err == nil {
				t.Fatal("expected command to fail")
			}

			formattedErr := formatGitError(err, testURL)
			if !strings.Contains(formattedErr.Error(), tt.expectedContain) {
				t.Errorf("formatGitError() = %q, expected to contain %q", formattedErr.Error(), tt.expectedContain)
			}
		})
	}
}

func TestFormatGitError_NonExitError(t *testing.T) {
	// Test with a non-ExitError (should return original error)
	originalErr := os.ErrNotExist
	result := formatGitError(originalErr, "https://example.com")

	if result != originalErr {
		t.Errorf("formatGitError with non-ExitError should return original error, got %v", result)
	}
}

func TestGitCloneArgs(t *testing.T) {
	cloneURL := "https://github.com/golang/go"
	destPath := "/tmp/go"

	got := gitCloneArgs(cloneURL, destPath)
	want := []string{
		"clone",
		"--progress",
		"--single-branch",
		"--depth", "1",
		cloneURL,
		destPath,
	}

	if len(got) != len(want) {
		t.Fatalf("gitCloneArgs() len = %d, want %d (%v)", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("gitCloneArgs()[%d] = %q, want %q (full args: %v)", i, got[i], want[i], got)
		}
	}
}

func TestParseGitProgress(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expectedStage string
		expectedLine  string
		expectedOK    bool
	}{
		{
			name:          "receiving objects",
			line:          "Receiving objects:  42% (42/100), 1.23 MiB | 1.23 MiB/s",
			expectedStage: "Receiving objects",
			expectedLine:  "Receiving objects: 42% (42/100), 1.23 MiB | 1.23 MiB/s",
			expectedOK:    true,
		},
		{
			name:          "remote counting objects",
			line:          "remote: Counting objects: 100% (10/10), done.",
			expectedStage: "Counting objects",
			expectedLine:  "Counting objects: 100% (10/10), done.",
			expectedOK:    true,
		},
		{
			name:       "non progress line",
			line:       "Cloning into '/tmp/repo'...",
			expectedOK: false,
		},
		{
			name:       "fatal error line",
			line:       "fatal: repository 'https://example.com/repo' not found",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stage, line, ok := parseGitProgress(tt.line)
			if ok != tt.expectedOK {
				t.Fatalf("parseGitProgress(%q) ok = %v, expected %v", tt.line, ok, tt.expectedOK)
			}

			if !tt.expectedOK {
				return
			}

			if stage != tt.expectedStage || line != tt.expectedLine {
				t.Fatalf(
					"parseGitProgress(%q) = (%q, %q), expected (%q, %q)",
					tt.line,
					stage,
					line,
					tt.expectedStage,
					tt.expectedLine,
				)
			}
		})
	}
}

func TestGitProgressWriterFormatsStages(t *testing.T) {
	var output bytes.Buffer
	writer := newGitProgressWriter(&output)

	chunks := []string{
		"Cloning into '/tmp/repo'...\n",
		"remote: Counting objects:  50% (1/2)\rremote: Counting objects: 100% (2/2), done.\r",
		"Receiving objects:  25% (1/4)\rReceiving objects: 100% (4/4), done.\r",
		"Resolving deltas: 100% (1/1), done.\n",
	}

	for _, chunk := range chunks {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatalf("writer.Write() error = %v", err)
		}
	}

	if err := writer.Finish(); err != nil {
		t.Fatalf("writer.Finish() error = %v", err)
	}

	got := output.String()
	expectedParts := []string{
		"Cloning into '/tmp/repo'...\n",
		"\rCounting objects: 50% (1/2)",
		"\rCounting objects: 100% (2/2), done.",
		"\n\rReceiving objects: 25% (1/4)",
		"\rReceiving objects: 100% (4/4), done.",
		"\n\rResolving deltas: 100% (1/1), done.\n",
	}

	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Fatalf("progress output %q does not contain %q", got, part)
		}
	}
}

func TestGitProgressWriterEndsProgressBeforeErrors(t *testing.T) {
	var output bytes.Buffer
	writer := newGitProgressWriter(&output)

	input := "Receiving objects: 100% (1/1), done.\rfatal: repository 'https://example.com/repo' not found\n"
	if _, err := writer.Write([]byte(input)); err != nil {
		t.Fatalf("writer.Write() error = %v", err)
	}

	if err := writer.Finish(); err != nil {
		t.Fatalf("writer.Finish() error = %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "\rReceiving objects: 100% (1/1), done.\nfatal: repository 'https://example.com/repo' not found\n") {
		t.Fatalf("progress output %q did not preserve the fatal error after finishing the progress line", got)
	}
}
