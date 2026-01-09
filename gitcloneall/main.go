package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	configFileName = ".clone-all-config.json"
	defaultWorkers = 8
	defaultLimit   = 1000
	defaultBranch  = "master"
)

type Config struct {
	DefaultOrg    string `json:"default_org"`
	DefaultBranch string `json:"default_branch"`
	Workers       int    `json:"workers"`
}

type Repo struct {
	Name             string `json:"name"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

type CloneResult struct {
	RepoName string
	Status   string // "cloned", "skipped", "failed"
	Error    error
	Duration time.Duration
}

func main() {
	// Subcommands
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	cloneCmd := flag.NewFlagSet("clone", flag.ExitOnError)
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)

	// Init flags
	initOrg := initCmd.String("org", "", "Default organization to clone from")
	initBranch := initCmd.String("branch", defaultBranch, "Default branch to checkout")
	initWorkers := initCmd.Int("workers", defaultWorkers, "Number of concurrent workers")

	// Clone flags
	cloneOrg := cloneCmd.String("org", "", "Organization to clone from (overrides default)")
	cloneBranch := cloneCmd.String("branch", "", "Branch to checkout (overrides default)")
	cloneWorkers := cloneCmd.Int("workers", 0, "Number of concurrent workers (overrides default)")
	cloneLimit := cloneCmd.Int("limit", defaultLimit, "Maximum number of repos to fetch")
	cloneDir := cloneCmd.String("dir", ".", "Directory to clone repos into")
	cloneDryRun := cloneCmd.Bool("dry-run", false, "Show what would be done without doing it")
	cloneForce := cloneCmd.Bool("force", false, "Re-clone repos that already exist (removes existing)")
	cloneSSH := cloneCmd.Bool("ssh", true, "Use SSH for cloning (default: true)")
	cloneFilter := cloneCmd.String("filter", "", "Filter repos by name (substring match)")

	// Config flags
	configShow := configCmd.Bool("show", false, "Show current configuration")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		handleInit(*initOrg, *initBranch, *initWorkers)
	case "clone":
		cloneCmd.Parse(os.Args[2:])
		handleClone(*cloneOrg, *cloneBranch, *cloneWorkers, *cloneLimit, *cloneDir, *cloneDryRun, *cloneForce, *cloneSSH, *cloneFilter)
	case "config":
		configCmd.Parse(os.Args[2:])
		if *configShow {
			handleConfigShow()
		} else {
			configCmd.Usage()
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`clone-all - Clone all repositories from a GitHub organization

USAGE:
    clone-all <command> [options]

COMMANDS:
    init     Initialize configuration with default organization
    clone    Clone repositories from an organization
    config   View or modify configuration
    help     Show this help message

EXAMPLES:
    # Set up default organization
    clone-all init -org mycompany

    # Clone all repos from default org
    clone-all clone

    # Clone with custom settings
    clone-all clone -org otherorg -workers 16 -dir ./repos

    # Dry run to see what would be cloned
    clone-all clone -dry-run

    # Clone only repos matching a filter
    clone-all clone -filter api

REQUIREMENTS:
    - GitHub CLI (gh) must be installed and authenticated
    - Git must be installed`)
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return configFileName
	}
	return filepath.Join(home, configFileName)
}

func loadConfig() (*Config, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				DefaultBranch: defaultBranch,
				Workers:       defaultWorkers,
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults for missing values
	if config.Workers == 0 {
		config.Workers = defaultWorkers
	}
	if config.DefaultBranch == "" {
		config.DefaultBranch = defaultBranch
	}

	return &config, nil
}

func saveConfig(config *Config) error {
	configPath := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func handleInit(org, branch string, workers int) {
	if org == "" {
		fmt.Print("Enter default organization: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		org = strings.TrimSpace(input)
	}

	if org == "" {
		fmt.Println("Error: organization is required")
		os.Exit(1)
	}

	config := &Config{
		DefaultOrg:    org,
		DefaultBranch: branch,
		Workers:       workers,
	}

	if err := saveConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Configuration saved to %s\n", getConfigPath())
	fmt.Printf("  Default org:    %s\n", config.DefaultOrg)
	fmt.Printf("  Default branch: %s\n", config.DefaultBranch)
	fmt.Printf("  Workers:        %d\n", config.Workers)
}

func handleConfigShow() {
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📋 Configuration (%s):\n", getConfigPath())
	fmt.Printf("  Default org:    %s\n", config.DefaultOrg)
	fmt.Printf("  Default branch: %s\n", config.DefaultBranch)
	fmt.Printf("  Workers:        %d\n", config.Workers)
}

func handleClone(org, branch string, workers, limit int, dir string, dryRun, force, useSSH bool, filter string) {
	// Check prerequisites
	if err := checkPrerequisites(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// Load config and apply overrides
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	if org == "" {
		org = config.DefaultOrg
	}
	if org == "" {
		fmt.Fprintf(os.Stderr, "❌ No organization specified. Use -org flag or run 'clone-all init' first.\n")
		os.Exit(1)
	}

	if branch == "" {
		branch = config.DefaultBranch
	}
	if workers == 0 {
		workers = config.Workers
	}

	// Ensure target directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error creating directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	// Change to target directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error changing to directory %s: %v\n", dir, err)
		os.Exit(1)
	}
	defer os.Chdir(originalDir)

	fmt.Printf("🔄 Fetching repository list from %s...\n", org)

	// Fetch repos
	repos, err := fetchRepos(org, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching repos: %v\n", err)
		os.Exit(1)
	}

	// Apply filter if specified
	if filter != "" {
		var filtered []Repo
		for _, repo := range repos {
			if strings.Contains(strings.ToLower(repo.Name), strings.ToLower(filter)) {
				filtered = append(filtered, repo)
			}
		}
		repos = filtered
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found matching criteria.")
		return
	}

	fmt.Printf("📂 Found %d repositories in %s\n", len(repos), org)
	fmt.Printf("🔧 Using %d workers\n\n", workers)

	if dryRun {
		fmt.Println("🔍 Dry run mode - would process:")
		for _, repo := range repos {
			exists := dirExists(repo.Name)
			status := "clone"
			if exists && !force {
				status = "skip (exists)"
			} else if exists && force {
				status = "force re-clone"
			}
			fmt.Printf("   • %s [%s]\n", repo.Name, status)
		}
		return
	}

	// Clone repos concurrently
	results := cloneReposConcurrently(repos, org, branch, workers, force, useSSH)

	// Print summary
	printSummary(results)
}

func checkPrerequisites() error {
	// Check for gh CLI
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) not found. Please install it: https://cli.github.com/")
	}

	// Check for git
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found. Please install git")
	}

	// Check if gh is authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub CLI not authenticated. Run 'gh auth login' first")
	}

	return nil
}

func fetchRepos(org string, limit int) ([]Repo, error) {
	cmd := exec.Command("gh", "repo", "list", org,
		"--limit", fmt.Sprintf("%d", limit),
		"--json", "name,defaultBranchRef")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var repos []Repo
	if err := json.Unmarshal(output, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repo list: %w", err)
	}

	return repos, nil
}

func cloneReposConcurrently(repos []Repo, org, defaultBranch string, workers int, force, useSSH bool) []CloneResult {
	// Limit workers to number of CPUs if not specified higher
	if workers > runtime.NumCPU()*2 {
		workers = runtime.NumCPU() * 2
	}
	if workers > len(repos) {
		workers = len(repos)
	}

	jobs := make(chan Repo, len(repos))
	results := make(chan CloneResult, len(repos))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobs {
				result := cloneRepo(repo, org, defaultBranch, force, useSSH)
				printProgress(result)
				results <- result
			}
		}()
	}

	// Send jobs
	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results
	var allResults []CloneResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}

func cloneRepo(repo Repo, org, defaultBranch string, force, useSSH bool) CloneResult {
	start := time.Now()
	result := CloneResult{RepoName: repo.Name}

	// Check if directory exists
	if dirExists(repo.Name) {
		if !force {
			result.Status = "skipped"
			result.Duration = time.Since(start)
			return result
		}

		// Remove existing directory for force re-clone
		if err := os.RemoveAll(repo.Name); err != nil {
			result.Status = "failed"
			result.Error = fmt.Errorf("failed to remove existing directory: %w", err)
			result.Duration = time.Since(start)
			return result
		}
	}

	// Determine clone URL
	var cloneURL string
	if useSSH {
		cloneURL = fmt.Sprintf("git@github.com:%s/%s.git", org, repo.Name)
	} else {
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git", org, repo.Name)
	}

	// Determine branch to checkout
	branch := repo.DefaultBranchRef.Name
	if branch == "" {
		branch = defaultBranch
	}

	// Clone with specific branch
	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", cloneURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without branch specification (repo might not have the branch)
		cmd = exec.Command("git", "clone", cloneURL)
		output, err = cmd.CombinedOutput()
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Errorf("clone failed: %s", strings.TrimSpace(string(output)))
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Status = "cloned"
	result.Duration = time.Since(start)
	return result
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func printProgress(result CloneResult) {
	switch result.Status {
	case "cloned":
		fmt.Printf("✅ %s: cloned (%s)\n", result.RepoName, result.Duration.Round(time.Second))
	case "skipped":
		fmt.Printf("☑️  %s: skipped (already exists)\n", result.RepoName)
	case "failed":
		fmt.Printf("❌ %s: %v\n", result.RepoName, result.Error)
	}
}

func printSummary(results []CloneResult) {
	var cloned, skipped, failed int
	var failedRepos []CloneResult
	var skippedRepos []CloneResult

	for _, r := range results {
		switch r.Status {
		case "cloned":
			cloned++
		case "skipped":
			skipped++
			skippedRepos = append(skippedRepos, r)
		case "failed":
			failed++
			failedRepos = append(failedRepos, r)
		}
	}

	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Println("📊 Summary")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("✅ Cloned:  %d\n", cloned)
	fmt.Printf("☑️  Skipped: %d\n", skipped)
	fmt.Printf("❌ Failed:  %d\n", failed)
	fmt.Printf("📁 Total:   %d\n", len(results))

	// if len(skippedRepos) > 0 {
	// 	fmt.Println("\n☑️  Skipped repositories (already exist):")
	// 	for _, r := range skippedRepos {
	// 		fmt.Printf("   • %s\n", r.RepoName)
	// 	}
	// }

	if len(failedRepos) > 0 {
		fmt.Println("\n❌ Failed repositories:")
		for _, r := range failedRepos {
			fmt.Printf("   • %s: %v\n", r.RepoName, r.Error)
		}
		os.Exit(1)
	}
}
