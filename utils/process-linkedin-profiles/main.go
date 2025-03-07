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

// Configuration struct to hold settings
type Config struct {
	InputFolder   string
	OutputFolder  string
	LogFolder     string
	LogFile       string
	MaxWorkers    int
	Verbose       bool
	FabricCommand string // New field for fabric command
}

func main() {
	// Define command-line flags
	config := Config{}
	flag.StringVar(&config.InputFolder, "input", "data/test/split", "Path to the folder containing input JSON files")
	flag.StringVar(&config.OutputFolder, "output", "data/test/profile", "Path to the folder where processed LinkedIn profiles will be saved")
	flag.StringVar(&config.LogFolder, "logdir", "logs", "Folder for storing log files")
	flag.IntVar(&config.MaxWorkers, "workers", 5, "Maximum number of concurrent workers")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&config.FabricCommand, "fabric-cmd", "summarize_linkedin_profile", "Fabric command to use for processing")
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

	// Get all JSON files
	jsonFiles, err := filepath.Glob(filepath.Join(config.InputFolder, "*.json"))
	if err != nil {
		message := fmt.Sprintf("ERROR: Failed to read JSON files: %v", err)
		logAndPrint(logger, message, config.Verbose)
		os.Exit(1)
	}

	// Check if any JSON files were found
	if len(jsonFiles) == 0 {
		message := fmt.Sprintf("WARNING: No JSON files found in %s", config.InputFolder)
		logAndPrint(logger, message, config.Verbose)
		os.Exit(0)
	} else {
		message := fmt.Sprintf("INFO: Found %d JSON files to process", len(jsonFiles))
		logAndPrint(logger, message, config.Verbose)
	}

	// Create worker pool for parallel processing
	var wg sync.WaitGroup
	var mutex sync.Mutex // For thread-safe logging
	semaphore := make(chan struct{}, config.MaxWorkers)

	// Process each JSON file
	for _, file := range jsonFiles {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a token
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the token when done
			processJSONFile(filePath, config, logger, &mutex)
		}(file)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Log completion
	logAndPrint(logger, fmt.Sprintf("INFO: Processing completed. Processed %d LinkedIn profiles.", len(jsonFiles)), config.Verbose)
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

// Process a single JSON file
func processJSONFile(filePath string, config Config, logger *log.Logger, mutex *sync.Mutex) {
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outputFilePath := filepath.Join(config.OutputFolder, fileNameWithoutExt+".md")

	if config.Verbose {
		fmt.Printf("Processing LinkedIn profile JSON: %s\n", filePath)
		fmt.Printf("Input file: %s\n", filePath)
		fmt.Printf("Output file: %s\n", outputFilePath)
		fmt.Printf("Using fabric command: %s\n", config.FabricCommand)
	}

	// Read the content of the input file
	content, err := os.ReadFile(filePath)
	if err != nil {
		message := fmt.Sprintf("ERROR: Failed to read file %s - %v", filePath, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		return
	}

	// Create a command to run fabric with the specified command
	cmd := exec.Command("fabric", "-p", config.FabricCommand, "-o", outputFilePath)

	// Create stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		message := fmt.Sprintf("ERROR: Failed to create stdin pipe for fabric command - %v", err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
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
		return
	}

	// Write content to stdin and close it
	if _, err := stdin.Write(content); err != nil {
		message := fmt.Sprintf("ERROR: Failed to write to fabric stdin for %s - %v", filePath, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		return
	}
	stdin.Close()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		message := fmt.Sprintf("ERROR: Failed to process LinkedIn profile '%s' with command '%s'. Error: %v", filePath, config.FabricCommand, err)
		logMessage(logger, message, mutex)
		fmt.Println(message)
		return
	}

	message := fmt.Sprintf("SUCCESS: Processed LinkedIn profile '%s' successfully with command '%s'.", filePath, config.FabricCommand)
	logMessage(logger, message, mutex)
	if config.Verbose {
		fmt.Println(message)
	} else {
		fmt.Printf("Processed: %s\n", fileNameWithoutExt)
	}
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
