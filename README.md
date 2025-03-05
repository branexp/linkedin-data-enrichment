# linkedin-data-enrichment
This repository uses Golang and Powershell to make LLM API calls to enrich data with Linkedin profile URLs.

A comprehensive toolkit for processing LinkedIn profiles from JSONL data, generating summaries, and attaching them to CSV files.

## Overview

This toolkit provides a set of utilities for a complete LinkedIn profile processing workflow:

1. Split JSONL files containing LinkedIn profiles into individual JSON files
2. Process the LinkedIn profiles to generate markdown summaries
3. Attach these summaries to a CSV file for further analysis or reporting

The toolkit was designed to facilitate data processing for recruitment, market research, networking, or other professional activities that involve analyzing LinkedIn profiles.

## Components

The toolkit consists of three main components:

### 1. JSONL Splitter

A Go utility that splits JSONL files into individual JSON files, using the LinkedIn profile's public identifier as the filename.

### 2. Profile Processor

A PowerShell script that processes LinkedIn profile JSON files using the 'fabric' tool to generate markdown summaries.

### 3. CSV Profile Attacher

A Go utility that matches the generated profile summaries to rows in a CSV file and attaches them as a new column.

## Prerequisites

- **Go** (1.16 or later) for the Go utilities
- **PowerShell** (5.1 or later) for the profile processing script
- **Fabric** tool (must be installed and available in your PATH)

## Installation

1. Clone this repository:
   ```bash
   git clone https://github.com/yourusername/linkedin-profile-toolkit.git
   cd linkedin-profile-toolkit
   ```

2. Build the Go utilities:
   ```bash
   cd utils/jsonl-splitter
   go build -o jsonl-splitter
   cd ../csv-profile-attacher
   go build -o csv-profile-attacher
   ```

## Usage

### 1. Split JSONL File

```bash
./utils/jsonl-splitter/jsonl-splitter -input data/your-profiles.jsonl -output data/test/split -pretty
```

Options:
- `-input`: Path to the JSONL file (required)
- `-output`: Directory to store the output JSON files (default: "output")
- `-fallback-prefix`: Prefix for output filenames when publicIdentifier is not found (default: "item")
- `-pretty`: Format JSON with indentation for readability

### 2. Process LinkedIn Profiles

```powershell
./scripts/processLinkedinProfiles.ps1
```

This script uses the following default paths:
- Input folder: `data\test\split`
- Output folder: `data\test\profile`
- Log file: `logs\profile_process.log`

You can modify these paths directly in the script if needed.

### 3. Attach Profiles to CSV

```bash
./utils/csv-profile-attacher/csv-profile-attacher -csv data/your-data.csv -profiles data/test/profile -column linkedin_profile_summary
```

Options:
- `-csv`: Path to the CSV file (default: "data/test/csv/data.csv")
- `-profiles`: Directory containing markdown profiles (default: "data/test/profile")
- `-output`: Output CSV file path (defaults to overwriting input CSV)
- `-column`: Name of the column to add/update (default: "linkedin_profile_summary")
- `-verbose`: Enable verbose logging

## Complete Workflow Example

1. Place your LinkedIn profiles JSONL file in the `data` directory.

2. Split the JSONL file into individual JSON files:
   ```bash
   ./utils/jsonl-splitter/jsonl-splitter -input data/linkedin-profiles.jsonl -output data/test/split -pretty
   ```

3. Process the LinkedIn profiles to generate markdown summaries:
   ```powershell
   ./scripts/processLinkedinProfiles.ps1
   ```

4. Attach the summaries to your CSV file:
   ```bash
   ./utils/csv-profile-attacher/csv-profile-attacher -csv data/contacts.csv -profiles data/test/profile -output data/enriched-contacts.csv
   ```

## Project Structure

```
├── cmd/                   # Reserved for future command-line tools
├── data/
│   └── test/              # Test data directories
│       ├── csv/           # CSV files
│       ├── jsonl/         # Original JSONL files
│       ├── profile/       # Generated markdown profiles
│       └── split/         # Split JSON files
├── logs/                  # Log files
├── scripts/
│   └── processLinkedinProfiles.ps1  # Profile processing script
└── utils/
    ├── csv-profile-attacher/       # Utility to attach profiles to CSV
    │   └── main.go
    └── jsonl-splitter/             # Utility to split JSONL files
        └── main.go
```

## Error Handling

- All components include error handling and logging
- The PowerShell script logs processing details to `logs/profile_process.log`
- The Go utilities output progress and error information to the console

## Notes

- The `fabric` tool is required for profile processing but is not included in this repository. Ensure it's installed and available in your PATH.
- The toolkit assumes specific data structures; you may need to modify the code if your LinkedIn data has a different format.