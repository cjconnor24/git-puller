package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	// Define command line flags
	baseDir := flag.String("basedir", "", "Base directory to search for Git repositories (defaults to current directory)")
	operation := flag.String("operation", "pull", "Git operation to perform: pull, fetch, or fetch-prune")
	flag.Parse()

	// Determine the directory to use
	var currentDir string
	var err error

	if *baseDir != "" {
		// Validate that the provided basedir exists
		if _, err := os.Stat(*baseDir); err != nil {
			fmt.Printf("Error: basedir path does not exist: %s\n", *baseDir)
			os.Exit(1)
		}
		currentDir = *baseDir
	} else {
		// Get current working directory
		currentDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Searching for Git repositories in: %s\n", currentDir)

	// Find all Git repositories
	gitRepos, err := findGitRepos(currentDir)
	if err != nil {
		fmt.Printf("Error finding Git repositories: %v\n", err)
		os.Exit(1)
	}

	if len(gitRepos) == 0 {
		fmt.Println("No Git repositories found.")
		return
	}

	fmt.Printf("Found %d Git repositories:\n", len(gitRepos))
	for _, repo := range gitRepos {
		fmt.Printf("  - %s\n", repo)
	}

	fmt.Printf("\nPerforming %q on repositories...\n", *operation)

	// Create a channel for repos and a WaitGroup for synchronization
	repoChan := make(chan string, len(gitRepos))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errorRepos []string
	numWorkers := 32
	activeCount := 0
	startTime := time.Now()

	// Start worker goroutines
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, repoChan, &wg, &mu, &activeCount, &errorRepos, *operation)
	}

	// Send repos to the channel
	for _, repo := range gitRepos {
		repoChan <- repo
	}
	close(repoChan)

	// Wait for all workers to complete
	wg.Wait()

	elapsed := time.Since(startTime)
	fmt.Printf("\nUpdate complete! Total time: %.2f seconds\n", elapsed.Seconds())

	// Print error summary
	if len(errorRepos) > 0 {
		fmt.Printf("\n❌ The following %d repositories failed to update:\n", len(errorRepos))
		for _, repo := range errorRepos {
			fmt.Printf("  - %s\n", repo)
		}
	} else {
		fmt.Println("\n✅ All repositories updated successfully.")
	}
}

// findGitRepos finds all directories containing .git folders at depth 1
func findGitRepos(baseDir string) ([]string, error) {
	var gitRepos []string

	// Read directories in current directory
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(baseDir, entry.Name())
			gitPath := filepath.Join(dirPath, ".git")

			// Check if .git exists (either as directory or file for worktrees)
			if _, err := os.Stat(gitPath); err == nil {
				gitRepos = append(gitRepos, dirPath)
			}
		}
	}

	return gitRepos, nil
}

// updateRepo runs the git operation in the specified repository
func updateRepo(repoName string, mu *sync.Mutex, activeCount *int, errorRepos *[]string, operation string) {
	mu.Lock()
	*activeCount++
	currentActive := *activeCount
	mu.Unlock()

	fmt.Printf("[%d active] --- %s %s ---\n", currentActive, strings.Title(operation), repoName)

	var cmd *exec.Cmd
	switch operation {
	case "fetch":
		cmd = exec.Command("git", "-C", repoName, "fetch")
	case "fetch-prune":
		cmd = exec.Command("git", "-C", repoName, "fetch", "--prune")
	default:
		cmd = exec.Command("git", "-C", repoName, "pull")
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		mu.Lock()
		fmt.Printf("❌ Error %s %s: %v\n", operation, repoName, err)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", strings.TrimSpace(string(output)))
		}
		*errorRepos = append(*errorRepos, repoName)
		*activeCount--
		mu.Unlock()
	} else {
		mu.Lock()
		fmt.Printf("✅ Successfully %s %s\n", operation, repoName)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", strings.TrimSpace(string(output)))
		}
		*activeCount--
		mu.Unlock()
	}
}

// worker processes repos from the channel
func worker(id int, repos <-chan string, wg *sync.WaitGroup, mu *sync.Mutex, activeCount *int, errorRepos *[]string, operation string) {
	defer wg.Done()

	for repo := range repos {
		updateRepo(repo, mu, activeCount, errorRepos, operation)
	}
}
