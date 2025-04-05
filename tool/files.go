package tool

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileListInput defines the input parameters for the Files tool
type FileListInput struct {
	Path      string `json:"path" description:"The directory path to list files from"`
	Recursive bool   `json:"recursive" description:"Whether to list files recursively (including subdirectories)"`
	Pattern   string `json:"pattern" description:"Optional glob pattern to filter files (e.g., *.go, *.md)"`
}

func ListFiles(input FileListInput) string {
	if input.Path == "" {
		return "Error: Path is required"
	}

	// Check if the directory exists
	info, err := os.Stat(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if !info.IsDir() {
		return fmt.Sprintf("Error: %s is not a directory", input.Path)
	}

	var files []string
	var walkErr error

	// List files based on recursive flag
	if input.Recursive {
		// Walk through all subdirectories
		walkErr = filepath.Walk(input.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories themselves unless it's the root
			if info.IsDir() && path != input.Path {
				return nil
			}

			// Check pattern if provided
			if input.Pattern != "" {
				match, err := filepath.Match(input.Pattern, filepath.Base(path))
				if err != nil {
					return err
				}
				if !match {
					return nil
				}
			}

			// Add the file to our list
			if !info.IsDir() {
				files = append(files, path)
			}

			return nil
		})
	} else {
		// Non-recursive listing
		entries, err := os.ReadDir(input.Path)
		if err != nil {
			return fmt.Sprintf("Error reading directory: %v", err)
		}

		for _, entry := range entries {
			// Skip directories if we're not including them
			if entry.IsDir() {
				continue
			}

			filename := entry.Name()

			// Check pattern if provided
			if input.Pattern != "" {
				match, err := filepath.Match(input.Pattern, filename)
				if err != nil {
					return fmt.Sprintf("Error with pattern matching: %v", err)
				}
				if !match {
					continue
				}
			}

			files = append(files, filepath.Join(input.Path, filename))
		}
	}

	if walkErr != nil {
		return fmt.Sprintf("Error walking directory: %v", walkErr)
	}

	// Format output
	if len(files) == 0 {
		return "No files found matching the criteria"
	}

	result := fmt.Sprintf("Found %d files:\n", len(files))
	for _, file := range files {
		result += fmt.Sprintf("- %s\n", file)
	}

	return result
}
