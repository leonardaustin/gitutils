package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Result struct {
	Dir     string
	Success bool
	Message string
	Error   error
}

func main() {
	// Flags
	maxWorkers := flag.Int("workers", 8, "maximum number of concurrent git operations")
	timeout := flag.Duration("timeout", 5*time.Minute, "timeout per repository")
	verbose := flag.Bool("verbose", false, "show detailed output")
	dryRun := flag.Bool("dry-run", false, "show what would be done without executing")
	targetDir := flag.String("dir", ".", "directory containing git repositories")
	flag.Parse()

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⚠️  Received interrupt signal, gracefully shutting down...")
		cancel()
	}()

	// Change to target directory
	absTargetDir, err := filepath.Abs(*targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error resolving directory: %v\n", err)
		os.Exit(1)
	}

	// Find all git repositories
	repos, err := findGitRepos(absTargetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error finding repositories: %v\n", err)
		os.Exit(1)
	}

	if len(repos) == 0 {
		fmt.Println("No git repositories found in", absTargetDir)
		os.Exit(0)
	}

	fmt.Printf("📂 Found %d git repositories in %s\n", len(repos), absTargetDir)
	fmt.Printf("🔧 Using %d workers with %v timeout per repo\n\n", *maxWorkers, *timeout)

	if *dryRun {
		fmt.Println("🔍 Dry run mode - would process:")
		for _, repo := range repos {
			fmt.Printf("   • %s\n", filepath.Base(repo))
		}
		os.Exit(0)
	}

	// Process repositories concurrently
	results := processRepos(ctx, repos, *maxWorkers, *timeout, *verbose)

	// Print summary
	printSummary(results)
}

func findGitRepos(baseDir string) ([]string, error) {
	var repos []string

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		dirPath := filepath.Join(baseDir, entry.Name())
		gitPath := filepath.Join(dirPath, ".git")

		// Check if .git exists (either as directory or file for worktrees)
		if _, err := os.Stat(gitPath); err == nil {
			repos = append(repos, dirPath)
		}
	}

	return repos, nil
}

func processRepos(ctx context.Context, repos []string, maxWorkers int, timeout time.Duration, verbose bool) []Result {
	var wg sync.WaitGroup
	results := make([]Result, len(repos))
	semaphore := make(chan struct{}, maxWorkers)

	for i, repo := range repos {
		wg.Add(1)
		go func(idx int, repoPath string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[idx] = Result{
					Dir:     filepath.Base(repoPath),
					Success: false,
					Message: "cancelled",
					Error:   ctx.Err(),
				}
				return
			}

			results[idx] = processRepo(ctx, repoPath, timeout, verbose)
		}(i, repo)
	}

	wg.Wait()
	return results
}

func processRepo(ctx context.Context, repoPath string, timeout time.Duration, verbose bool) Result {
	repoName := filepath.Base(repoPath)
	result := Result{Dir: repoName}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check if repository has any commits (empty repos have no HEAD)
	_, err := runGitCommand(ctx, repoPath, "rev-parse", "HEAD")
	if err != nil {
		// Check if this is an empty repository (no commits yet)
		result.Success = true
		result.Message = "skipped (empty repository - no commits yet)"
		fmt.Printf("⚠️  %s: %s\n", repoName, result.Message)
		return result
	}

	// Get current branch
	branch, err := runGitCommand(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		result.Success = false
		result.Message = "not a valid git repository"
		result.Error = err
		fmt.Printf("❌ %s: %s\n", repoName, result.Message)
		return result
	}
	branch = strings.TrimSpace(branch)

	// Check for uncommitted changes
	status, err := runGitCommand(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		result.Success = false
		result.Message = "failed to check status"
		result.Error = err
		fmt.Printf("❌ %s: %s\n", repoName, result.Message)
		return result
	}

	hasChanges := len(strings.TrimSpace(status)) > 0

	// Fetch from remote
	if verbose {
		fmt.Printf("🔄 %s: fetching...\n", repoName)
	}

	_, err = runGitCommand(ctx, repoPath, "fetch", "--all", "--prune")
	if err != nil {
		result.Success = false
		result.Message = "fetch failed"
		result.Error = err
		fmt.Printf("❌ %s: %s - %v\n", repoName, result.Message, err)
		return result
	}

	// Check if we have an upstream configured
	upstream, err := runGitCommand(ctx, repoPath, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err != nil {
		result.Success = true
		result.Message = fmt.Sprintf("fetched (no upstream for %s)", branch)
		fmt.Printf("⚠️  %s: %s\n", repoName, result.Message)
		return result
	}
	upstream = strings.TrimSpace(upstream)

	// Check if we're behind
	behindCount, err := runGitCommand(ctx, repoPath, "rev-list", "--count", fmt.Sprintf("HEAD..%s", upstream))
	if err != nil {
		result.Success = false
		result.Message = "failed to check upstream status"
		result.Error = err
		fmt.Printf("❌ %s: %s\n", repoName, result.Message)
		return result
	}

	behind := strings.TrimSpace(behindCount)
	if behind == "0" {
		result.Success = true
		result.Message = fmt.Sprintf("already up to date (%s)", branch)
		fmt.Printf("✅ %s: %s\n", repoName, result.Message)
		return result
	}

	// If there are uncommitted changes, don't pull
	if hasChanges {
		result.Success = true
		result.Message = fmt.Sprintf("fetched only - %s commits behind (uncommitted changes)", behind)
		fmt.Printf("⚠️  %s: %s\n", repoName, result.Message)
		return result
	}

	// Pull changes
	if verbose {
		fmt.Printf("🔄 %s: pulling %s commits...\n", repoName, behind)
	}

	output, err := runGitCommand(ctx, repoPath, "pull", "--ff-only")
	if err != nil {
		// Try rebase if ff-only fails
		output, err = runGitCommand(ctx, repoPath, "pull", "--rebase")
		if err != nil {
			result.Success = false
			result.Message = "pull failed (tried ff-only and rebase)"
			result.Error = err
			fmt.Printf("❌ %s: %s - %v\n", repoName, result.Message, err)
			return result
		}
	}

	result.Success = true
	if strings.Contains(output, "Already up to date") {
		result.Message = fmt.Sprintf("up to date (%s)", branch)
	} else {
		result.Message = fmt.Sprintf("pulled %s commits (%s)", behind, branch)
	}
	fmt.Printf("✅ %s: %s\n", repoName, result.Message)

	return result
}

func runGitCommand(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out")
		}
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("command cancelled")
		}
		return string(output), fmt.Errorf("%w: %s", err, string(output))
	}

	return string(output), nil
}

func printSummary(results []Result) {
	var successful, failed, warnings int

	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Println("📊 Summary")
	fmt.Println(strings.Repeat("─", 50))

	var failedRepos []Result
	var warningRepos []Result

	for _, r := range results {
		if r.Success {
			if strings.Contains(r.Message, "uncommitted") || strings.Contains(r.Message, "no upstream") || strings.Contains(r.Message, "empty repository") {
				warnings++
				warningRepos = append(warningRepos, r)
			} else {
				successful++
			}
		} else {
			failed++
			failedRepos = append(failedRepos, r)
		}
	}

	fmt.Printf("✅ Successful: %d\n", successful)
	fmt.Printf("⚠️  Warnings:   %d\n", warnings)
	fmt.Printf("❌ Failed:     %d\n", failed)
	fmt.Printf("📁 Total:      %d\n", len(results))

	if len(warningRepos) > 0 {
		fmt.Println("\n⚠️  Repositories with warnings:")
		for _, r := range warningRepos {
			fmt.Printf("   • %s: %s\n", r.Dir, r.Message)
		}
	}

	if len(failedRepos) > 0 {
		fmt.Println("\n❌ Failed repositories:")
		for _, r := range failedRepos {
			fmt.Printf("   • %s: %s\n", r.Dir, r.Message)
			if r.Error != nil {
				fmt.Printf("     Error: %v\n", r.Error)
			}
		}
		os.Exit(1)
	}
}
