package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// File types supported by the processor
const (
	FileTypeJSON     = "json"
	FileTypeMarkdown = "md"
	FileTypeUnknown  = "unknown"
)

// Configuration struct to hold settings
type Config struct {
	InputFolder   string
	OutputFolder  string
	LogFolder     string
	LogFile       string
	MaxWorkers    int
	Verbose       bool
	FabricCommand string // Field for fabric command with optional arguments
}

// ProcessingStats tracks statistics about the processing
type ProcessingStats struct {
	Total      int
	Successful int
	Failed     int
	Skipped    int
	JSONFiles  int
	MDFiles    int
}

// Initialize a new ProcessingStats
func newProcessingStats() *ProcessingStats {
	return &ProcessingStats{}
}

// Increment the successful count and file type count
func (s *ProcessingStats) incrementSuccessful(mutex *sync.Mutex, fileType string) {
	mutex.Lock()
	defer mutex.Unlock()
	s.Successful++
	if fileType == FileTypeJSON {
		s.JSONFiles++
	} else if fileType == FileTypeMarkdown {
		s.MDFiles++
	}
}

// Increment the failed count
func (s *ProcessingStats) incrementFailed(mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
	s.Failed++
}

// Increment the skipped count
func (s *ProcessingStats) incrementSkipped(mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
	s.Skipped++
}

// Set the total count
func (s *ProcessingStats) setTotal(total int) {
	s.Total = total
}

// Get a summary string
func (s *ProcessingStats) getSummary() string {
	return fmt.Sprintf(
		"Total: %d, Successful: %d (JSON: %d, MD: %d), Failed: %d, Skipped: %d",
		s.Total, s.Successful, s.JSONFiles, s.MDFiles, s.Failed, s.Skipped,
	)
}

func main() {
	// Define command-line flags
	config := Config{}
	flag.StringVar(&config.InputFolder, "input", "data/test/split", "Path to the folder containing input JSON and markdown files")
	flag.StringVar(&config.OutputFolder, "output", "data/test/profile", "Path to the folder where processed profiles will be saved")
	flag.StringVar(&config.LogFolder, "logdir", "logs", "Folder for storing log files")
	flag.IntVar(&config.MaxWorkers, "workers", 5, "Maximum number of concurrent workers")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&config.FabricCommand, "fabric-cmd", "summarize_linkedin_profile",
		"Fabric command with optional arguments (e.g., 'summarize_linkedin_profile -t 0.7')")
	flag.Parse()

	// Set log file path
	config.LogFile = filepath.Join(config.LogFolder, "profile_process.log")

	// Ensure directories exist
	ensureDirectoryExists(config.OutputFolder)
	ensureDirectoryExists(config.LogFolder)

	// Initialize log file
	logFile := initLogFile(config.LogFile)
	defer logFile.Close()

	// Set up logger
	logger := log.New(logFile, "", 0)

	// Log the configuration
	logAndPrint(logger, fmt.Sprintf("INFO: Using fabric command: %s", config.FabricCommand), config.Verbose)

	// Get all input files (JSON and markdown)
	inputFiles, err := findInputFiles(config.InputFolder)
	if err != nil {
		message := fmt.Sprintf("ERROR: Failed to read input files: %v", err)
		logAndPrint(logger, message, config.Verbose)
		os.Exit(1)
	}

	// Check if any files were found
	if len(inputFiles) == 0 {
		message := fmt.Sprintf("WARNING: No JSON or markdown files found in %s", config.InputFolder)
		logAndPrint(logger, message, config.Verbose)
		os.Exit(0)
	} else {
		message := fmt.Sprintf("INFO: Found %d files to process", len(inputFiles))
		logAndPrint(logger, message, config.Verbose)
	}

	// Create worker pool for parallel processing
	var wg sync.WaitGroup
	var mutex sync.Mutex // For thread-safe logging
	semaphore := make(chan struct{}, config.MaxWorkers)
	stats := newProcessingStats()
	stats.setTotal(len(inputFiles))

	// Process each file
	for _, file := range inputFiles {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a token
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the token when done
			processFile(filePath, config, logger, &mutex, stats)
		}(file)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Log completion with statistics
	completionMsg := fmt.Sprintf("INFO: Processing completed. %s", stats.getSummary())
	logAndPrint(logger, completionMsg, config.Verbose)
}

// ParseFabricCommand parses a fabric command string into command name and arguments
func parseFabricCommand(cmdString string) (string, []string) {
	parts := strings.Fields(cmdString)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// Find all input files (JSON and markdown)
func findInputFiles(inputFolder string) ([]string, error) {
	var allFiles []string

	// Find JSON files
	jsonFiles, err := filepath.Glob(filepath.Join(inputFolder, "*.json"))
	if err != nil {
		return nil, err
	}
	allFiles = append(allFiles, jsonFiles...)

	// Find markdown files
	mdFiles, err := filepath.Glob(filepath.Join(inputFolder, "*.md"))
	if err != nil {
		return nil, err
	}
	allFiles = append(allFiles, mdFiles...)

	return allFiles, nil
}

// Detect the file type based on file extension
func detectFileType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return FileTypeJSON
	case ".md":
		return FileTypeMarkdown
	default:
		return FileTypeUnknown
	}
}

// Ensure a directory exists, creating it if necessary
func ensureDirectoryExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Failed to create directory: %s - %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}
}

// Initialize the log file
func initLogFile(logFilePath string) *os.File {
	// Remove existing log file if it exists
	if _, err := os.Stat(logFilePath); err == nil {
		if err := os.Remove(logFilePath); err != nil {
			fmt.Printf("Failed to remove existing log file: %v\n", err)
			os.Exit(1)
		}
	}

	// Create new log file
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Printf("Failed to create log file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized log file: %s\n", logFilePath)
	return logFile
}

// Process a single file (JSON or markdown)
func processFile(filePath string, config Config, logger *log.Logger, mutex *sync.Mutex, stats *ProcessingStats) {
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outputFilePath := filepath.Join(config.OutputFolder, fileNameWithoutExt+".md")
	fileType := detectFileType(filePath)

	// Parse the fabric command into base command and arguments
	cmdName, cmdArgs := parseFabricCommand(config.FabricCommand)

	if cmdName == "" {
		message := "ERROR: Empty fabric command specified"
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementFailed(mutex)
		return
	}

	// Log file processing information
	if config.Verbose {
		fmt.Printf("Processing file: %s (type: %s)\n", filePath, fileType)
		fmt.Printf("Input file: %s\n", filePath)
		fmt.Printf("Output file: %s\n", outputFilePath)
		fmt.Printf("Using fabric command: %s with args: %v\n", cmdName, cmdArgs)
	}

	// Skip unknown file types
	if fileType == FileTypeUnknown {
		message := fmt.Sprintf("WARNING: Skipping file with unknown type: %s", filePath)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementSkipped(mutex)
		return
	}

	// Read the content of the input file
	content, err := os.ReadFile(filePath)
	if err != nil {
		message := fmt.Sprintf("ERROR: Failed to read file %s - %v", filePath, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementFailed(mutex)
		return
	}

	// Create the fabric command with appropriate arguments
	fabArgs := append([]string{"-p", cmdName}, cmdArgs...)
	fabArgs = append(fabArgs, "-o", outputFilePath)

	cmd := exec.Command("fabric", fabArgs...)

	if config.Verbose {
		fmt.Printf("Executing command: fabric %s\n", strings.Join(fabArgs, " "))
	}

	// Create stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		message := fmt.Sprintf("ERROR: Failed to create stdin pipe for fabric command - %v", err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementFailed(mutex)
		return
	}

	// Redirect stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		message := fmt.Sprintf("ERROR: Failed to start fabric command '%s' for %s - %v", config.FabricCommand, filePath, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementFailed(mutex)
		return
	}

	// Write content to stdin and close it
	if _, err := stdin.Write(content); err != nil {
		message := fmt.Sprintf("ERROR: Failed to write to fabric stdin for %s - %v", filePath, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementFailed(mutex)
		return
	}
	stdin.Close()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		message := fmt.Sprintf("ERROR: Failed to process file '%s' with command '%s'. Error: %v", filePath, config.FabricCommand, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		stats.incrementFailed(mutex)
		return
	}

	message := fmt.Sprintf("SUCCESS: Processed file '%s' (type: %s) successfully with command '%s'.", filePath, fileType, config.FabricCommand)
	logMessage(logger, message, mutex)
	if config.Verbose {
		fmt.Println(message)
	} else {
		fmt.Printf("Processed: %s (%s)\n", fileNameWithoutExt, fileType)
	}

	// Update statistics
	stats.incrementSuccessful(mutex, fileType)
}

// Log a message to the log file
func logMessage(logger *log.Logger, message string, mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()

	timestamp := time.Now().Format(time.RFC3339)
	logger.Println(timestamp + " - " + message)
}

// Log a message and optionally print it
func logAndPrint(logger *log.Logger, message string, verbose bool) {
	timestamp := time.Now().Format(time.RFC3339)
	logger.Println(timestamp + " - " + message)
	if verbose {
		fmt.Println(message)
	} else {
		// Print important messages even in non-verbose mode
		if strings.HasPrefix(message, "INFO:") || strings.HasPrefix(message, "WARNING:") {
			fmt.Println(message)
		}
	}
}
