package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
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

	fmt.Println("\nUpdating repositories...")

	// Update each repository
	for _, repo := range gitRepos {
		updateRepo(repo)
	}

	fmt.Println("\nUpdate complete!")
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
				gitRepos = append(gitRepos, entry.Name())
			}
		}
	}

	return gitRepos, nil
}

// updateRepo runs git pull in the specified repository
func updateRepo(repoName string) {
	fmt.Printf("\n--- Updating %s ---\n", repoName)

	cmd := exec.Command("git", "-C", repoName, "pull")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("❌ Error updating %s: %v\n", repoName, err)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", strings.TrimSpace(string(output)))
		}
	} else {
		fmt.Printf("✅ Successfully updated %s\n", repoName)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", strings.TrimSpace(string(output)))
		}
	}
}