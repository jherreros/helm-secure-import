package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func main() {
	config, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(config *Config) error {
	if config.DryRun {
		fmt.Println("🔍 DRY RUN MODE - No changes will be made")
		fmt.Println("=====================================")
	}

	report := &Report{}

	// Resolve report file path to absolute path before changing directories
	if config.ReportFile != "" {
		absPath, err := filepath.Abs(config.ReportFile)
		if err != nil {
			return fmt.Errorf("failed to resolve report file path: %w", err)
		}
		config.ReportFile = absPath
		if config.DryRun {
			fmt.Printf("📝 Would write report to: %s\n", config.ReportFile)
		} else {
			fmt.Printf("📝 Report will be written to: %s\n", config.ReportFile)
		}
	}

	// Create temp directory for artifacts
	fmt.Println("🏗️  Setting up workspace...")
	tmpDir, err := os.MkdirTemp("", "helm-import-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		return fmt.Errorf("failed to change to temp directory: %w", err)
	}
	defer os.Chdir(originalDir)

	fmt.Printf("📦 Pulling chart %s:%s from %s...\n", config.ChartName, config.Version, config.Repo)
	if config.IsOCI {
		chartURL := fmt.Sprintf("%s/%s", config.Repo, config.ChartName)
		if err := execCommand("helm", "pull", chartURL, "--version", config.Version); err != nil {
			return fmt.Errorf("failed to pull chart from OCI registry: %w", err)
		}
	} else {
		if err := execCommand("helm", "pull", config.ChartName, "--version", config.Version, "--repo", config.Repo); err != nil {
			return fmt.Errorf("failed to pull chart: %w", err)
		}
	}

	// Check if chart exists
	chartRef := fmt.Sprintf("%s/charts/%s:%s", config.Registry, config.ChartName, config.Version)
	report.Chart.Name = chartRef

	fmt.Printf("🔍 Checking if chart exists in registry: %s\n", chartRef)
	chartExists := false
	chartExists, err = imageExists(chartRef)
	if err != nil {
		return err
	}

	if !chartExists {
		if config.DryRun {
			fmt.Printf("📤 Would push and sign chart: %s\n", chartRef)
		} else {
			fmt.Printf("📤 Pushing and signing chart: %s\n", chartRef)
			if err := pushAndSignChart(config); err != nil {
				return err
			}
		}
		report.Chart.Pushed = true
	} else {
		fmt.Printf("✅ Chart %s:%s already exists. Skipping push.\n", config.ChartName, config.Version)
	}

	// Get images from chart
	fmt.Println("🔍 Extracting container images from chart...")
	images, err := getImagesFromChart(config)
	if err != nil {
		return fmt.Errorf("failed to extract images from chart: %w", err)
	}

	if len(images) == 0 {
		fmt.Println("ℹ️  No container images found in chart")
	} else {
		fmt.Printf("🎯 Found %d container image(s) to process:\n", len(images))
		for _, image := range images {
			fmt.Printf("  • %s\n", image)
		}
	}

	// Process each image
	if len(images) > 0 {
		fmt.Printf("\n� Processing %d images in parallel...\n", len(images))
		imageReports := processImagesInParallel(images, config)
		report.Images = imageReports
	}

	fmt.Println("\n📊 Generating report...")
	return report.GenerateReport(config.ReportFormat, config.ReportFile, config)
}

// processImagesInParallel processes multiple images concurrently using a worker pool
func processImagesInParallel(images []string, config *Config) []ImageReport {
	// Use a reasonable number of workers (max 4, or number of CPUs if less)
	numWorkers := runtime.NumCPU()
	if numWorkers > 4 {
		numWorkers = 4
	}
	if len(images) < numWorkers {
		numWorkers = len(images)
	}

	// Channels for coordination
	imageChan := make(chan string, len(images))
	resultChan := make(chan ImageReport, len(images))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for image := range imageChan {
				fmt.Printf("🔄 Worker %d processing: %s\n", workerID+1, image)

				imageReport := ImageReport{Name: image}

				if config.DryRun {
					fmt.Printf("  🔍 [Worker %d] Would check if image exists in registry\n", workerID+1)
					fmt.Printf("  🛡️  [Worker %d] Would scan for vulnerabilities\n", workerID+1)
					fmt.Printf("  🔧 [Worker %d] Would patch if vulnerabilities found\n", workerID+1)
					fmt.Printf("  📤 [Worker %d] Would push image to registry\n", workerID+1)
					fmt.Printf("  ✍️  [Worker %d] Would sign image\n", workerID+1)
					imageReport.Pushed = true
					imageReport.VulnerabilitiesFound = 0
					imageReport.Patched = false
					imageReport.Signed = false
				} else {
					pushed, err := processImage(image, config)
					if err != nil {
						fmt.Printf("❌ [Worker %d] Failed to process image %s: %v\n", workerID+1, image, err)
						imageReport.Pushed = false
					} else {
						imageReport.Pushed = pushed
						// TODO: Get actual vulnerability count, patch status, and sign status from processImage
						imageReport.VulnerabilitiesFound = 0
						imageReport.Patched = false
						imageReport.Signed = config.Sign
					}
				}

				resultChan <- imageReport
			}
		}(i)
	}

	// Send images to workers
	for _, image := range images {
		imageChan <- image
	}
	close(imageChan)

	// Wait for all workers to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	var results []ImageReport
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}
