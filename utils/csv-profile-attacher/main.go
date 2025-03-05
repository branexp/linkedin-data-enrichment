package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Define command-line flags
	csvPath := flag.String("csv", "data/test/csv/data.csv", "Path to the CSV file")
	profileDir := flag.String("profiles", "data/test/profile", "Directory containing markdown profiles")
	outputCSV := flag.String("output", "", "Output CSV file path (defaults to overwriting input CSV)")
	columnName := flag.String("column", "linkedin_profile_summary", "Name of the column to add/update")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Configure logging
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	log.Printf("Processing CSV file: %s", *csvPath)
	log.Printf("Profile directory: %s", *profileDir)

	// If no output path specified, use the input path
	if *outputCSV == "" {
		*outputCSV = *csvPath
	}
	log.Printf("Output will be written to: %s", *outputCSV)

	// Read the CSV file
	csvFile, err := os.Open(*csvPath)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	// Parse the CSV
	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error reading CSV: %v\n", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Println("CSV file is empty")
		os.Exit(1)
	}

	log.Printf("Read %d rows from CSV file", len(records))

	// Find or add the profile summary column
	headers := records[0]
	profileColIndex := -1
	for i, header := range headers {
		if header == *columnName {
			profileColIndex = i
			log.Printf("Found existing column '%s' at index %d", *columnName, i)
			break
		}
	}

	// If column doesn't exist, add it
	if profileColIndex == -1 {
		headers = append(headers, *columnName)
		profileColIndex = len(headers) - 1
		records[0] = headers
		log.Printf("Added new column '%s' at index %d", *columnName, profileColIndex)

		// Add empty column value to all existing rows
		for i := 1; i < len(records); i++ {
			if len(records[i]) < len(headers) {
				records[i] = append(records[i], "")
			}
		}
	}

	// Read profile markdown files
	profileFiles, err := os.ReadDir(*profileDir)
	if err != nil {
		fmt.Printf("Error reading profile directory: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Found %d files in profile directory", len(profileFiles))

	// Track statistics
	attachedCount := 0
	notFoundCount := 0

	// Process each markdown file
	for _, file := range profileFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			// Extract base filename without extension
			baseFilename := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			log.Printf("Processing profile: %s", baseFilename)

			// Read markdown content
			mdContent, err := os.ReadFile(filepath.Join(*profileDir, file.Name()))
			if err != nil {
				fmt.Printf("Error reading markdown file %s: %v\n", file.Name(), err)
				continue
			}

			// Find matching row in CSV
			matched := false
			for i := 1; i < len(records); i++ {
				// Check each field in the row for the profile identifier
				for j, field := range records[i] {
					if strings.Contains(field, baseFilename) {
						// Ensure the row has enough columns
						for len(records[i]) <= profileColIndex {
							records[i] = append(records[i], "")
						}

						// Update the row with the profile content
						records[i][profileColIndex] = string(mdContent)

						log.Printf("Found match in row %d, column %d", i, j)
						fmt.Printf("Attached profile for %s\n", baseFilename)
						matched = true
						attachedCount++
						break
					}
				}
				if matched {
					break
				}
			}

			if !matched {
				fmt.Printf("Could not find matching row for profile %s\n", baseFilename)
				notFoundCount++
			}
		}
	}

	// Write the updated CSV
	outputFile, err := os.Create(*outputCSV)
	if err != nil {
		fmt.Printf("Error creating output CSV file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)

	// Configure the writer to handle CSV fields properly
	writer.UseCRLF = true // Use Windows-style line endings for better compatibility

	// Write all records
	err = writer.WriteAll(records)
	if err != nil {
		fmt.Printf("Error writing CSV: %v\n", err)
		os.Exit(1)
	}
	writer.Flush()

	if err := writer.Error(); err != nil {
		fmt.Printf("Error flushing CSV writer: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("CSV update summary:\n")
	fmt.Printf("- Profiles attached: %d\n", attachedCount)
	fmt.Printf("- Profiles not found: %d\n", notFoundCount)
	fmt.Printf("Successfully updated CSV with profile summaries at %s\n", *outputCSV)
}
