package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type NodeData struct {
	Verr struct {
		Properties struct {
			File string `json:"file"`
			Line int    `json:"line"`
		} `json:"properties"`
	} `json:"verr"`
}

type OutputEntry struct {
	Filename string      `json:"filename"`
	Line     interface{} `json:"line"`
}

func main() {
	var inputFile = flag.String("input", "", "Path to input JSONL file")
	var relativeRoot = flag.String("root", "", "Relative path root to strip from filenames")
	var outputFolder = flag.String("output", "", "Output folder for generated JSON files")

	flag.Parse()

	if *inputFile == "" || *relativeRoot == "" || *outputFolder == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <input.jsonl> -root <relative_root> -output <output_folder>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -input nodes.jsonl -root '/Users/byron/repos/third-party/injective' -output ./annotations\n", os.Args[0])
		os.Exit(1)
	}

	// Ensure relative root ends with separator for proper stripping
	if !strings.HasSuffix(*relativeRoot, "/") {
		*relativeRoot += "/"
	}

	err := processJSONL(*inputFile, *relativeRoot, *outputFolder)
	if err != nil {
		log.Fatalf("Error processing file: %v", err)
	}

	fmt.Printf("Successfully processed %s\n", *inputFile)
}

func processJSONL(inputFile, relativeRoot, outputFolder string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// Track files we've written to, so we can manage appending vs creating
	writtenFiles := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	nextId := 0

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse the entire JSON line
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			log.Printf("Warning: Failed to parse JSON on line %d: %v", nextId, err)
			continue
		}

		// Extract the filename from the nested structure
		filename, err := extractFilename(jsonData)
		if err != nil {
			log.Printf("Warning: Failed to extract filename from line %d: %v", nextId, err)
			continue
		}

		// Strip the relative root from the filename
		if !strings.HasPrefix(filename, relativeRoot) {
			log.Printf("Warning: File %s does not start with root %s, skipping", filename, relativeRoot)
			continue
		}

		relativePath := strings.TrimPrefix(filename, relativeRoot)
		relativePath = strings.TrimPrefix(relativePath, "/")

		// Create output file path
		outputFilePath := filepath.Join(outputFolder, relativePath+".annotations.json")

		// Create the output entry
		outputEntry := OutputEntry{
			Filename: filename,
			Line:     jsonData,
		}

		// Write to output file
		if err := writeToOutputFile(outputFilePath, outputEntry, !writtenFiles[outputFilePath]); err != nil {
			return fmt.Errorf("failed to write to output file %s: %w", outputFilePath, err)
		}

		writtenFiles[outputFilePath] = true
		nextId++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	return nil
}

func extractFilename(jsonData map[string]interface{}) (string, error) {
	// Navigate through the nested structure to find the file field
	verr, ok := jsonData["verr"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing or invalid 'verr' field")
	}

	properties, ok := verr["properties"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing or invalid 'properties' field")
	}

	file, ok := properties["file"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'file' field")
	}

	return file, nil
}

func writeToOutputFile(outputFilePath string, entry OutputEntry, isNewFile bool) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var entries []OutputEntry

	// If file exists, read existing entries
	if !isNewFile {
		if existingData, err := os.ReadFile(outputFilePath); err == nil {
			if err := json.Unmarshal(existingData, &entries); err != nil {
				return fmt.Errorf("failed to parse existing JSON file %s: %w", outputFilePath, err)
			}
		}
	}

	// Append new entry
	entries = append(entries, entry)

	// Write back to file
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputFilePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
