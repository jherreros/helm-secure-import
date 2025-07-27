package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	report := &Report{}

	// Resolve report file path to absolute path before changing directories
	if config.ReportFile != "" {
		absPath, err := filepath.Abs(config.ReportFile)
		if err != nil {
			return fmt.Errorf("failed to resolve report file path: %w", err)
		}
		config.ReportFile = absPath
		fmt.Printf("Report will be written to: %s\n", config.ReportFile)
	}

	// Create temp directory for artifacts
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

	// Pull Helm chart
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
	chartExists, err := imageExists(chartRef)
	if err != nil {
		return err
	}

	if !chartExists {
		if err := pushAndSignChart(config); err != nil {
			return err
		}
		report.Chart.Pushed = true
	} else {
		fmt.Printf("Chart %s:%s already exists. Skipping push.\n", config.ChartName, config.Version)
	}

	// Get images from chart
	images, err := getImagesFromChart(config)
	if err != nil {
		return err
	}

	// Process each image
	for _, image := range images {
		pushed, err := processImage(image, config)
		if err != nil {
			return fmt.Errorf("failed to process image %s: %w", image, err)
		}
		report.Images = append(report.Images, ImageReport{Name: image, Pushed: pushed})
	}

	return report.GenerateReport(config.ReportFormat, config.ReportFile)
}