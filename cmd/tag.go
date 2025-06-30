package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Version type flag
	versionType string
)

// tagCmd represents the tag command
var tagCmd = &cobra.Command{
	Use:   "tag [message]",
	Short: "Create a new Git tag and push it to remote repository",
	Long: `Create a new Git tag and push it to remote repository.
If there is no remote tag, v0.0.1 will be used as the initial version.
If there are existing tags, the version number will be incremented based on the latest tag.
You can specify the version increment type with the -v flag:
  - major: Increment major version (v1.0.0 -> v2.0.0)
  - minor: Increment minor version (v1.0.0 -> v1.1.0)
  - patch: Increment patch version (v1.0.0 -> v1.0.1)`,
	// Change Args to accept 0 or 1 argument
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Message is optional now
		message := ""
		if len(args) > 0 {
			message = args[0]
		}
		err := createAndPushTag(message, versionType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	// Add version type flag
	tagCmd.Flags().StringVarP(&versionType, "version", "v", "patch", "Version update type (major|minor|patch)")
}

// Create and push tag
func createAndPushTag(message string, versionType string) error {
	// 1. Get the latest remote tag
	fmt.Println("Fetching remote tags...")
	latestTag, err := getLatestTag()
	if err != nil {
		fmt.Println("Failed to get remote tags:", err)
		fmt.Println("Will use default version v0.0.1")
		latestTag = "v0.0.1"
	} else {
		fmt.Printf("Latest tag found: %s\n", latestTag)
	}

	// 2. Generate new tag version
	newTag, err := incrementVersion(latestTag, versionType)
	if err != nil {
		return fmt.Errorf("failed to increment version: %v", err)
	}
	fmt.Printf("New tag version: %s\n", newTag)

	// 3. Create new tag
	fmt.Printf("Creating tag %s...\n", newTag)
	err = createTag(newTag, message)
	if err != nil {
		return fmt.Errorf("failed to create tag: %v", err)
	}

	// 4. Push tag to remote repository
	fmt.Printf("Pushing tag %s to remote repository...\n", newTag)
	err = pushTag(newTag)
	if err != nil {
		return fmt.Errorf("failed to push tag: %v", err)
	}

	fmt.Printf("Tag %s has been successfully created and pushed to remote repository\n", newTag)
	return nil
}

// Get the latest remote tag
func getLatestTag() (string, error) {
	// First get all tags
	cmd := exec.Command("git", "ls-remote", "--tags", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// If no tags, return empty string
	if len(output) == 0 {
		return "", fmt.Errorf("no remote tags found")
	}

	// Parse output and extract tags
	tags := []string{}
	lines := strings.Split(string(output), "\n")
	pattern := regexp.MustCompile(`refs/tags/v(\d+\.\d+\.\d+)$`)

	for _, line := range lines {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			tags = append(tags, "v"+matches[1])
		}
	}

	// If no matching tags, return error
	if len(tags) == 0 {
		return "", fmt.Errorf("no semantic versioned remote tags found")
	}

	// Find the latest version tag
	latestTag := tags[0]
	for _, tag := range tags[1:] {
		// Compare versions
		if compareVersions(tag, latestTag) > 0 {
			latestTag = tag
		}
	}

	return latestTag, nil
}

// Compare two versions, return 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	v1Parts := extractVersionParts(v1)
	v2Parts := extractVersionParts(v2)

	for i := 0; i < 3; i++ {
		if v1Parts[i] > v2Parts[i] {
			return 1
		} else if v1Parts[i] < v2Parts[i] {
			return -1
		}
	}
	return 0
}

// Extract version parts
func extractVersionParts(version string) [3]int {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	result := [3]int{0, 0, 0}

	for i := 0; i < len(parts) && i < 3; i++ {
		val, err := strconv.Atoi(parts[i])
		if err == nil {
			result[i] = val
		}
	}

	return result
}

// Increment version number
func incrementVersion(version, versionType string) (string, error) {
	parts := extractVersionParts(version)

	switch strings.ToLower(versionType) {
	case "major":
		parts[0]++
		parts[1] = 0
		parts[2] = 0
	case "minor":
		parts[1]++
		parts[2] = 0
	case "patch":
		parts[2]++
	default:
		return "", fmt.Errorf("invalid version type: %s, should be major, minor, or patch", versionType)
	}

	return fmt.Sprintf("v%d.%d.%d", parts[0], parts[1], parts[2]), nil
}

// Create Git tag
func createTag(tag, message string) error {
	var cmd *exec.Cmd

	// If message is empty, use lightweight tag
	if message == "" {
		cmd = exec.Command("git", "tag", tag)
	} else {
		// If message is provided, create annotated tag
		cmd = exec.Command("git", "tag", "-a", tag, "-m", message)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Push tag to remote repository
func pushTag(tag string) error {
	cmd := exec.Command("git", "push", "origin", tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
