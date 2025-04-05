package tool

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

// CatInput defines the input parameters for the Cat tool
type CatInput struct {
	Path   string `json:"path" description:"The path of the file to read"`
	Offset int    `json:"offset" description:"Optional line number to start reading from (0-based)"`
	Limit  int    `json:"limit" description:"Optional maximum number of lines to read"`
}

func Cat(input CatInput) string {

	if input.Path == "" {
		return "Error: Path is required"
	}

	// Check if the file exists
	info, err := os.Stat(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if info.IsDir() {
		return fmt.Sprintf("Error: %s is a directory, not a file", input.Path)
	}

	file, err := os.Open(input.Path)
	if err != nil {
		return fmt.Sprintf("Error opening file: %v", err)
	}
	defer file.Close()

	// Use default values if not provided
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	limit := input.Limit
	if limit <= 0 {
		// Default to a reasonably large number if not specified
		limit = 1000
	}

	scanner := bufio.NewScanner(file)

	// Skip lines until we reach the offset
	lineCount := 0
	for lineCount < offset && scanner.Scan() {
		lineCount++
	}

	if scanner.Err() != nil {
		return fmt.Sprintf("Error reading file: %v", scanner.Err())
	}

	// If we reached EOF before the offset
	if lineCount < offset {
		return fmt.Sprintf("Error: File has only %d lines, offset %d is out of range", lineCount, offset)
	}

	// Read the requested lines
	var lines []string
	linesRead := 0
	for scanner.Scan() && linesRead < limit {
		lines = append(lines, scanner.Text())
		linesRead++
	}

	if scanner.Err() != nil {
		return fmt.Sprintf("Error reading file: %v", scanner.Err())
	}

	// Format the output with line numbers
	if len(lines) == 0 {
		if offset > 0 {
			return fmt.Sprintf("No lines to read after offset %d", offset)
		}
		return "File is empty"
	}

	result := fmt.Sprintf("File: %s (lines %d to %d)\n\n", input.Path, offset+1, offset+len(lines))

	// Add line numbers to output
	for i, line := range lines {
		lineNum := offset + i + 1
		// Ensure line numbers align nicely (right-justified)
		result += fmt.Sprintf("%5s | %s\n", strconv.Itoa(lineNum), line)
	}

	return result
}
