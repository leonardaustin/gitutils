package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessRepoCheckoutMainPullsMain(t *testing.T) {
	tmpDir := t.TempDir()
	seedRepo, repo := setupRemoteBackedRepo(t, tmpDir)

	runTestGit(t, repo, "checkout", "-b", "feature")

	if err := os.WriteFile(filepath.Join(seedRepo, "README.md"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, seedRepo, "add", "README.md")
	runTestGit(t, seedRepo, "commit", "-m", "update readme")
	runTestGit(t, seedRepo, "push")

	result := processRepo(context.Background(), repo, time.Minute, false, true)
	if !result.Success {
		t.Fatalf("processRepo failed: %s: %v", result.Message, result.Error)
	}

	branch := strings.TrimSpace(runTestGit(t, repo, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch != "main" {
		t.Fatalf("expected repo to be on main, got %q", branch)
	}

	content, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "v2\n" {
		t.Fatalf("expected main to be pulled to v2, got %q", string(content))
	}
}

func TestProcessRepoCheckoutMainSkipsDirtyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	_, repo := setupRemoteBackedRepo(t, tmpDir)

	runTestGit(t, repo, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := processRepo(context.Background(), repo, time.Minute, false, true)
	if !result.Success {
		t.Fatalf("processRepo failed: %s: %v", result.Message, result.Error)
	}
	if !strings.Contains(result.Message, "checkout main skipped") {
		t.Fatalf("expected dirty repo warning, got %q", result.Message)
	}

	branch := strings.TrimSpace(runTestGit(t, repo, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch != "feature" {
		t.Fatalf("expected dirty repo to stay on feature, got %q", branch)
	}
}

func setupRemoteBackedRepo(t *testing.T, tmpDir string) (string, string) {
	t.Helper()

	remote := filepath.Join(tmpDir, "origin.git")
	seedRepo := filepath.Join(tmpDir, "seed")
	repo := filepath.Join(tmpDir, "repo")

	runTestGit(t, tmpDir, "init", "--bare", "--initial-branch=main", remote)
	runTestGit(t, tmpDir, "init", "--initial-branch=main", seedRepo)

	if err := os.WriteFile(filepath.Join(seedRepo, "README.md"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, seedRepo, "add", "README.md")
	runTestGit(t, seedRepo, "commit", "-m", "initial commit")
	runTestGit(t, seedRepo, "remote", "add", "origin", remote)
	runTestGit(t, seedRepo, "push", "-u", "origin", "main")

	runTestGit(t, tmpDir, "clone", remote, repo)

	return seedRepo, repo
}

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	gitArgs := append([]string{
		"-c", "user.name=gitpullall test",
		"-c", "user.email=gitpullall@example.com",
	}, args...)

	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed in %s: %v\n%s", strings.Join(args, " "), dir, err, string(output))
	}

	return string(output)
}
