package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// readMarkdownFile reads a markdown file and extracts the headline (first line) and body (second line)
func readMarkdownFile(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("error opening markdown file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Get the headline (first line)
	var headline string
	if scanner.Scan() {
		headline = scanner.Text()
	} else {
		if scanner.Err() != nil {
			return "", "", fmt.Errorf("error reading headline: %w", scanner.Err())
		}
		// Empty file
		return "", "", nil
	}

	// Get the body (second line)
	var body string
	if scanner.Scan() {
		body = scanner.Text()
	} else {
		if scanner.Err() != nil {
			return "", "", fmt.Errorf("error reading body: %w", scanner.Err())
		}
		// Only one line in the file
		return headline, "", nil
	}

	return headline, body, nil
}

// findHeaderIndex finds the index of a header in a CSV header row, or adds it if not found
func findHeaderIndex(headers []string, columnName string) (int, []string, bool) {
	for i, header := range headers {
		if header == columnName {
			return i, headers, false
		}
	}
	// Header not found, add it
	return len(headers), append(headers, columnName), true
}

// findMatchingMarkdown searches for a markdown file that matches one of the CSV field values
func findMatchingMarkdown(messageDir string, csvRow []string, verbose bool) (string, bool) {
	files, err := os.ReadDir(messageDir)
	if err != nil {
		log.Printf("Error reading message directory: %v", err)
		return "", false
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		// Get the filename without extension for matching
		baseFilename := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		// Check if this filename matches any field in the CSV row
		for _, field := range csvRow {
			if strings.Contains(field, baseFilename) {
				if verbose {
					log.Printf("Found matching markdown file for %s: %s", field, file.Name())
				}
				return filepath.Join(messageDir, file.Name()), true
			}
		}
	}

	return "", false
}

func main() {
	// Define command-line flags
	csvPath := flag.String("csv", "data/test/csv/data.csv", "Path to the CSV file")
	messageDir := flag.String("messages", "data/test/message", "Directory containing markdown messages")
	outputCSV := flag.String("output", "", "Output CSV file path (defaults to overwriting input CSV)")
	headColumnName := flag.String("head", "headline", "Name of the headline column to add/update")
	bodyColumnName := flag.String("body", "body", "Name of the body column to add/update")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Configure logging
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	log.Printf("Processing CSV file: %s", *csvPath)
	log.Printf("Message directory: %s", *messageDir)

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

	// Find or add the headline and body columns
	headers := records[0]
	headColIndex, headers, headAdded := findHeaderIndex(headers, *headColumnName)
	bodyColIndex, headers, bodyAdded := findHeaderIndex(headers, *bodyColumnName)
	records[0] = headers

	if headAdded {
		log.Printf("Added new column '%s' at index %d", *headColumnName, headColIndex)
	} else {
		log.Printf("Found existing column '%s' at index %d", *headColumnName, headColIndex)
	}

	if bodyAdded {
		log.Printf("Added new column '%s' at index %d", *bodyColumnName, bodyColIndex)
	} else {
		log.Printf("Found existing column '%s' at index %d", *bodyColumnName, bodyColIndex)
	}

	// Add empty values to all existing rows if needed
	if headAdded || bodyAdded {
		for i := 1; i < len(records); i++ {
			for len(records[i]) < len(headers) {
				records[i] = append(records[i], "")
			}
		}
	}

	// Track statistics
	attachedCount := 0
	notFoundCount := 0

	// Process each row in the CSV
	for i := 1; i < len(records); i++ {
		// Ensure the row has enough columns
		for len(records[i]) < len(headers) {
			records[i] = append(records[i], "")
		}

		// Find matching markdown file
		mdPath, found := findMatchingMarkdown(*messageDir, records[i], *verbose)
		if !found {
			log.Printf("No matching markdown file found for row %d", i)
			notFoundCount++
			continue
		}

		// Read and parse the markdown file
		headline, body, err := readMarkdownFile(mdPath)
		if err != nil {
			log.Printf("Error reading markdown file %s: %v", mdPath, err)
			notFoundCount++
			continue
		}

		// Update the CSV row with headline and body
		records[i][headColIndex] = headline
		records[i][bodyColIndex] = body

		baseFilename := strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath))
		fmt.Printf("Attached headline and body for %s\n", baseFilename)
		attachedCount++
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
	fmt.Printf("Messages attached: %d\n", attachedCount)
	fmt.Printf("Messages not found: %d\n", notFoundCount)
	fmt.Printf("Successfully updated CSV with message headlines and bodies at %s\n", *outputCSV)
}
