# Parameter Definitions: Define variables for input/output folders and log file.
$inputFolder = "data\test\split"   # Path to the folder containing input JSON files.
$outputFolder = "data\test\profile"  # Path to the folder where processed LinkedIn profiles will be saved.
$logFolder = "logs"  # Folder for storing log files
$logFile = Join-Path -Path $logFolder -ChildPath "profile_process.log"  # Path to the file used for logging the process.

# Ensure the output folder exists: Check if the output directory exists, and create it if it doesn't.
if (-not (Test-Path $outputFolder)) {
    try {
        # Create the output directory. Stop on error. Suppress output.
        New-Item -ItemType Directory -Path $outputFolder -ErrorAction Stop | Out-Null
        Write-Host "Created output directory: $outputFolder" -ForegroundColor Green
    }
    catch {
        # Log the error and exit if creating the directory fails.
        Write-Error "Failed to create output directory: $($_.Exception.Message)"
        exit 1  # Exit with an error code.
    }
}

# Ensure the logs folder exists: Check if the logs directory exists, and create it if it doesn't.
if (-not (Test-Path $logFolder)) {
    try {
        # Create the logs directory. Stop on error. Suppress output.
        New-Item -ItemType Directory -Path $logFolder -ErrorAction Stop | Out-Null
        Write-Host "Created logs directory: $logFolder" -ForegroundColor Green
    }
    catch {
        # Log the error and exit if creating the directory fails.
        Write-Error "Failed to create logs directory: $($_.Exception.Message)"
        exit 1  # Exit with an error code.
    }
}

# Initialize the log file: Delete the log file if it exists, then create a new one.
try {
    if (Test-Path $logFile) {
        # Remove the existing log file, stopping on error.
        Remove-Item -Path $logFile -ErrorAction Stop
    }
    # Create a new log file, stopping on error, and suppressing output.
    New-Item -Path $logFile -ItemType File -ErrorAction Stop | Out-Null
    Write-Host "Initialized log file: $logFile" -ForegroundColor Green
}
catch {
    # If an error occurs during log file initialization, log it and exit.
    Write-Error "Failed to initialize log file: $($_.Exception.Message)"
    exit 1 # Exit with an error code.
}

# Create a lock object for thread-safe logging: This ensures that multiple threads can write to the log file without conflicts.
$logLock = New-Object System.Object

# Get all JSON files in the input folder: Retrieve all files with the .json extension in the specified input folder.
$jsonFiles = Get-ChildItem -Path $inputFolder -Filter '*.json' -ErrorAction Stop

# Check if any JSON files were found
if ($jsonFiles.Count -eq 0) {
    Write-Warning "No JSON files found in $inputFolder"
    Add-Content -Path $logFile -Value "$(Get-Date -Format 'u') - WARNING: No JSON files found in $inputFolder"
    exit 0
} else {
    Write-Host "Found $($jsonFiles.Count) JSON files to process" -ForegroundColor Green
    Add-Content -Path $logFile -Value "$(Get-Date -Format 'u') - INFO: Found $($jsonFiles.Count) JSON files to process"
}

# Define the script block for parallel processing: This script block will be executed for each JSON file.
$scriptBlock = {
    param (
        # Parameters passed to the script block.
        $file,          # The JSON file to process.
        $outputFolder,  # The output directory.
        $logFile,       # The log file path.
        $logLock        # The lock object for thread-safe logging.
    )

    # Function to handle thread-safe logging: Ensures only one thread writes to the log at a time.
    Function Write-Log {
        param (
            [string]$Message  # The message to log.
        )
        # Enter a critical section (thread-safe).
        [System.Threading.Monitor]::Enter($logLock)
        try {
            # Append the message to the log file.
            Add-Content -Path $logFile -Value $Message
        }
        finally {
            # Exit the critical section (thread-safe).
            [System.Threading.Monitor]::Exit($logLock)
        }
    }

    # Function to process each JSON file: Converts a single JSON file using the 'fabric' tool.
    Function Convert-JsonFile {
        param (
            [string]$InputFilePath,   # The full path to the input JSON file.
            [string]$OutputFilePath   # The full path to the output profile file.
        )
        try {
            # Read the content of the input file.
            $content = Get-Content -Path $InputFilePath -Raw -ErrorAction Stop
            # Process the content with 'fabric' and save it to the output file.
            $content | fabric -p summarize_linkedin_profile -o $OutputFilePath

            # Log a success message with timestamp.
            $successMessage = "$(Get-Date -Format 'u') - SUCCESS: Processed LinkedIn profile '$InputFilePath' successfully."
            Write-Log -Message $successMessage # Thread-safe logging
            Write-Host $successMessage -ForegroundColor Green
        }
        catch {
            # Log an error message with timestamp if processing fails.
            $errorMessage = "$(Get-Date -Format 'u') - ERROR: Failed to process LinkedIn profile '$InputFilePath'. Error: $($_.Exception.Message)"
            Write-Log -Message $errorMessage # Thread-safe logging.
            Write-Error $errorMessage
        }
    }

    # Diagnostic logging to ensure file properties are valid
    Write-Host "Processing LinkedIn profile JSON: $($file.FullName)"

    # Construct the full input and output file paths.
    $inputFile = $file.FullName  # Get the full path of the input file.
    $fileName = [System.IO.Path]::GetFileNameWithoutExtension($inputFile) # Get the file name without the extension.
    $outputFile = Join-Path -Path $outputFolder -ChildPath ("$fileName" + "_profile.md") # Create the output file path.

    # More diagnostic logging
    Write-Host "Input file: $inputFile"
    Write-Host "Output file: $outputFile"

    # Input validation: Check that input and output paths are not empty.
    if ([string]::IsNullOrWhiteSpace($inputFile) -or [string]::IsNullOrWhiteSpace($outputFile)) {
        # If either path is invalid, log an error and return (exit the script block).
        $errorMessage = "$(Get-Date -Format 'u') - ERROR: Input or output file path is empty. Input: '$inputFile', Output: '$outputFile'"
        Write-Log -Message $errorMessage # Thread safe logging.
        Write-Error $errorMessage
        return
    }

    # Call the Convert-JsonFile function to process the file.
    Convert-JsonFile -InputFilePath $inputFile -OutputFilePath $outputFile
}

# Process files in parallel using -AsJob: Start a job for each JSON file.
$jobs = @() # Initialize empty array.
foreach ($file in $jsonFiles) {
    # Start a new job for each file, passing the script block and arguments.
    $jobs += Start-Job -ScriptBlock $scriptBlock -ArgumentList $file, $outputFolder, $logFile, $logLock
}

# Wait for all jobs to complete and receive output.
$jobs | ForEach-Object { $_ | Wait-Job | Receive-Job }

# Log completion
$completionMessage = "$(Get-Date -Format 'u') - INFO: Processing completed. Processed $($jsonFiles.Count) LinkedIn profiles."
Add-Content -Path $logFile -Value $completionMessage
Write-Host $completionMessage -ForegroundColor Green
