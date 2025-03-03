package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Define command-line flags
	inputFile := flag.String("input", "", "Path to the JSONL file (required)")
	outputDir := flag.String("output", "output", "Directory to store the output JSON files")
	filePrefix := flag.String("prefix", "json", "Prefix for output filenames")
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

	// Process each line
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse JSON to verify it's valid
		var jsonData interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			fmt.Printf("Error parsing line %d: %v\n", lineCount, err)
			continue
		}

		// Create output filename
		outputFileName := filepath.Join(*outputDir, fmt.Sprintf("%s_%d.json", *filePrefix, lineCount))

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
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("Processed %d lines, created %d JSON files in %s\n", lineCount, successCount, *outputDir)
}
