package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Function to sanitize a string for use as a filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	sanitized := re.ReplaceAllString(name, "_")

	// Trim spaces from beginning and end
	sanitized = strings.TrimSpace(sanitized)

	// If empty after sanitization, return a default
	if sanitized == "" {
		return "item"
	}

	return sanitized
}

func main() {
	// Define command-line flags
	inputFile := flag.String("input", "", "Path to the JSONL file (required)")
	outputDir := flag.String("output", "output", "Directory to store the output JSON files")
	fallbackPrefix := flag.String("fallback-prefix", "item", "Prefix for output filenames when publicIdentifier is not found")
	prettyPrint := flag.Bool("pretty", false, "Format JSON with indentation for readability")
	flag.Parse()

	// Check if input file was provided
	if *inputFile == "" {
		fmt.Println("Error: Input file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Open input file
	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Prepare to scan file line by line
	scanner := bufio.NewScanner(file)
	lineCount := 0
	successCount := 0

	// Track used filenames to handle duplicates
	usedFilenames := make(map[string]int)

	// Process each line
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse JSON to verify it's valid
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			fmt.Printf("Error parsing line %d: %v\n", lineCount, err)
			continue
		}

		// Extract publicIdentifier or use fallback
		var prefix string
		if publicID, ok := jsonData["publicIdentifier"]; ok {
			if publicIDStr, isString := publicID.(string); isString {
				prefix = sanitizeFilename(publicIDStr)
			} else {
				prefix = fmt.Sprintf("%s_%d", *fallbackPrefix, lineCount)
			}
		} else {
			prefix = fmt.Sprintf("%s_%d", *fallbackPrefix, lineCount)
		}

		// Handle duplicate filenames by adding a counter
		basePrefix := prefix
		if count, exists := usedFilenames[basePrefix]; exists {
			count++
			usedFilenames[basePrefix] = count
			prefix = fmt.Sprintf("%s_%d", basePrefix, count)
		} else {
			usedFilenames[basePrefix] = 1
		}

		// Create output filename
		outputFileName := filepath.Join(*outputDir, fmt.Sprintf("%s.json", prefix))

		// Open output file
		outputFile, err := os.Create(outputFileName)
		if err != nil {
			fmt.Printf("Error creating output file for line %d: %v\n", lineCount, err)
			continue
		}

		// Write JSON to file
		var outputBytes []byte
		if *prettyPrint {
			// Format JSON with indentation for readability
			outputBytes, err = json.MarshalIndent(jsonData, "", "  ")
		} else {
			// Compact JSON format
			outputBytes, err = json.Marshal(jsonData)
		}

		if err != nil {
			fmt.Printf("Error converting line %d to JSON: %v\n", lineCount, err)
			outputFile.Close()
			continue
		}

		// Write to the output file
		if _, err := outputFile.Write(outputBytes); err != nil {
			fmt.Printf("Error writing line %d to file: %v\n", lineCount, err)
			outputFile.Close()
			continue
		}

		outputFile.Close()
		successCount++
		fmt.Printf("Created file: %s\n", outputFileName)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("Processed %d lines, created %d JSON files in %s\n", lineCount, successCount, *outputDir)
}
