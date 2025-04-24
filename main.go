package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drewwalton19216801/chronozip" // Adjust this import path if your module/directory structure is different
)

const (
	// Default extension for compressed files
	defaultExtension = ".cz"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [-d] [-o <outputfile>] <inputfile>\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  Compress:   %s data.txt\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  Compress:   %s -o data.compressed data.txt\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  Decompress: %s -d data.txt.cz\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(os.Stderr, "  Decompress: %s -d -o data.original data.txt.cz\n", filepath.Base(os.Args[0]))
}

func main() {
	// Define command-line flags
	decompressMode := flag.Bool("d", false, "Decompress mode (default is compress)")
	outputFile := flag.String("o", "", "Output file path (optional)")

	// Configure flag usage message
	flag.Usage = printUsage

	// Parse flags
	flag.Parse()

	// Check for required input file argument
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: Input file argument is required.\n\n")
		flag.Usage()
		os.Exit(1)
	}
	inputFile := flag.Arg(0)

	// --- Determine Input/Output Paths ---

	// Check if input file exists
	inputFileInfo, err := os.Stat(inputFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Input file not found: %s\n", inputFile)
		} else {
			fmt.Fprintf(os.Stderr, "Error accessing input file '%s': %v\n", inputFile, err)
		}
		os.Exit(1)
	}
	if inputFileInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: Input path is a directory, not a file: %s\n", inputFile)
		os.Exit(1)
	}

	// Determine output path if not specified
	if *outputFile == "" {
		if *decompressMode {
			// Decompress: Try removing default extension, otherwise add ".decompressed"
			if strings.HasSuffix(inputFile, defaultExtension) {
				*outputFile = strings.TrimSuffix(inputFile, defaultExtension)
			} else {
				*outputFile = inputFile + ".decompressed" // Avoid overwriting input if no extension match
			}
		} else {
			// Compress: Add default extension
			*outputFile = inputFile + defaultExtension
		}
	}

	// Prevent overwriting the input file unintentionally
	absInput, _ := filepath.Abs(inputFile)
	absOutput, _ := filepath.Abs(*outputFile)
	if absInput == absOutput {
		fmt.Fprintf(os.Stderr, "Error: Input and output file paths are the same: %s\n", absInput)
		os.Exit(1)
	}

	// --- Perform Action ---
	startTime := time.Now()

	if *decompressMode {
		fmt.Printf("Decompressing '%s' to '%s'...\n", inputFile, *outputFile)
		err = runDecompression(inputFile, *outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decompression failed: %v\n", err)
			// Clean up potentially partially written output file
			_ = os.Remove(*outputFile)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Compressing '%s' to '%s'...\n", inputFile, *outputFile)
		err = runCompression(inputFile, *outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compression failed: %v\n", err)
			// Clean up potentially partially written output file
			_ = os.Remove(*outputFile)
			os.Exit(1)
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("Success! Operation completed in %v.\n", duration)

	// Print size comparison (optional)
	if outputFileInfo, err := os.Stat(*outputFile); err == nil {
		inputSize := inputFileInfo.Size()
		outputSize := outputFileInfo.Size()
		ratio := float64(outputSize) / float64(inputSize) * 100
		if *decompressMode {
			ratio = float64(inputSize) / float64(outputSize) * 100 // Show ratio relative to original size
		}
		// Avoid division by zero if input size is 0
		if inputSize > 0 && !*decompressMode {
			fmt.Printf("Input Size: %d bytes, Output Size: %d bytes, Ratio: %.2f%%\n", inputSize, outputSize, ratio)
		} else if outputSize > 0 && *decompressMode {
			fmt.Printf("Input Size: %d bytes, Output Size: %d bytes\n", inputSize, outputSize)
		} else {
			fmt.Printf("Input Size: %d bytes, Output Size: %d bytes\n", inputSize, outputSize)
		}

	}
}

// runCompression handles the file opening, closing, and compression process.
func runCompression(inputFile, outputFile string) error {
	// Open input file
	r, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file '%s': %w", inputFile, err)
	}
	defer r.Close() // Ensure input file is closed

	// Create output file
	w, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %w", outputFile, err)
	}
	defer w.Close() // Ensure output file is closed

	// Create and run compressor
	compressor := chronozip.NewCompressor(w)
	err = compressor.Compress(r) // Compress handles flushing internally
	if err != nil {
		return fmt.Errorf("chronozip compression error: %w", err)
	}

	// Explicitly close the writer file handle here before returning (defer will also run)
	// to catch potential close errors, although Compress should have flushed everything.
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close output file '%s' after write: %w", outputFile, err)
	}

	return nil
}

// runDecompression handles the file opening, closing, and decompression process.
func runDecompression(inputFile, outputFile string) error {
	// Open input file (compressed data)
	r, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file '%s': %w", inputFile, err)
	}
	defer r.Close() // Ensure input file is closed

	// Create output file (decompressed data)
	w, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %w", outputFile, err)
	}
	defer w.Close() // Ensure output file is closed

	// Create and run decompressor
	decompressor := chronozip.NewDecompressor(r)
	err = decompressor.Decompress(w) // Decompress handles flushing its internal buffer
	if err != nil {
		return fmt.Errorf("chronozip decompression error: %w", err)
	}

	// Explicitly close the writer file handle (similar reasoning as in compression)
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close output file '%s' after write: %w", outputFile, err)
	}

	return nil
}
