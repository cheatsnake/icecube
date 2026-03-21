package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/processing"
	"github.com/cheatsnake/icecube/internal/service/processor"
)

func main() {
	var (
		inputFile    string
		outputFile   string
		outputFormat string
		maxDimension int
		quality      int
		keepMetadata bool
		showHelp     bool
	)

	flag.StringVar(&inputFile, "input", "", "Input image file path (required)")
	flag.StringVar(&inputFile, "i", "", "Input image file path (shorthand, required)")
	flag.StringVar(&outputFile, "output", "", "Output image file path (optional, defaults to input with format suffix)")
	flag.StringVar(&outputFile, "o", "", "Output image file path (shorthand)")
	flag.StringVar(&outputFormat, "format", "", "Output image format (jpg, png, webp, avif, etc.)")
	flag.StringVar(&outputFormat, "f", "", "Output image format (shorthand)")
	flag.IntVar(&maxDimension, "max-dimension", 0, "Maximum dimension (width or height), 0 means no resizing")
	flag.IntVar(&maxDimension, "d", 0, "Maximum dimension (shorthand)")
	flag.IntVar(&quality, "quality", 80, "Quality level (1-100, higher means better quality)")
	flag.IntVar(&quality, "q", 80, "Quality level (shorthand)")
	flag.BoolVar(&keepMetadata, "keep-metadata", false, "Keep metadata from original image")
	flag.BoolVar(&keepMetadata, "m", false, "Keep metadata (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Image Compression CLI Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -input <input-file> [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -input photo.jpg -format webp -max-dimension 1000\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i photo.png -f jpg -d 800 -c 90 -m\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s photo.webp -o compressed.jpg\n", os.Args[0])
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	if inputFile == "" && flag.NArg() > 0 {
		inputFile = flag.Arg(0)
	}

	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Input file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		log.Fatalf("Error: Input file '%s' does not exist\n", inputFile)
	}

	if outputFile == "" {
		ext := path.Ext(inputFile)
		baseName := strings.TrimSuffix(path.Base(inputFile), ext)
		dir := path.Dir(inputFile)
		if outputFormat != "" {
			outputFile = path.Join(dir, baseName+"."+strings.ToLower(outputFormat))
		} else {
			outputFile = path.Join(dir, baseName+"_processed"+ext)
		}
	}

	originalFormat := image.Format(strings.TrimPrefix(path.Ext(inputFile), "."))
	targetFormat := originalFormat
	if outputFormat != "" {
		targetFormat = image.Format(strings.ToLower(outputFormat))
	}

	start := time.Now()

	service, err := processor.NewService(slog.Default())
	if err != nil {
		log.Fatalf("Error creating service: %v\n", err)
	}

	opts, err := processing.NewOptions(
		targetFormat,
		maxDimension,
		quality,
		keepMetadata,
		make(map[string]string),
	)
	if err != nil {
		log.Fatalf("Error creating options: %v\n", err)
	}

	resultPath, err := service.Process(inputFile, opts)
	if err != nil {
		log.Fatalf("Error processing file %s: %v\n", inputFile, err)
	}

	if resultPath != outputFile {
		if err := os.Rename(resultPath, outputFile); err != nil {
			log.Fatalf("Error moving result to %s: %v\n", outputFile, err)
		}
		resultPath = outputFile
	}

	fileInfo, err := os.Stat(resultPath)
	if err != nil {
		log.Fatalf("Error getting output file info: %v\n", err)
	}

	inputInfo, err := os.Stat(inputFile)
	var inputSize int64
	if err != nil {
		log.Printf("Warning: could not get input file info: %v", err)
	} else {
		inputSize = inputInfo.Size()
	}

	fmt.Printf("\n✅ Processing completed successfully!\n")
	fmt.Printf("   Input:  %s (%s)\n", inputFile, formatFileSize(inputSize))
	fmt.Printf("   Output: %s (%s)\n", resultPath, formatFileSize(fileInfo.Size()))

	if inputSize > 0 {
		reduction := float64(inputSize-fileInfo.Size()) / float64(inputSize) * 100
		if reduction > 0 {
			fmt.Printf("   Size reduction: %.1f%%\n", reduction)
		} else if reduction < 0 {
			fmt.Printf("   Size increase: %.1f%%\n", -reduction)
		} else {
			fmt.Printf("   Size unchanged\n")
		}
	}

	fmt.Printf("   Time: %v\n", time.Since(start))
}

// formatFileSize formats file size in human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
