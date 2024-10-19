package main

import (
	"fmt"
	"os"
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

	// Pull Helm chart
	if err := execCommand("helm", "pull", config.ChartName, "--version", config.Version, "--repo", config.RepoURL); err != nil {
		return fmt.Errorf("failed to pull chart: %w", err)
	}

	// Check if chart exists in
	chartExists, err := checkChartExists(config)
	if err != nil {
		return err
	}

	if !chartExists {
		if err := pushAndSignChart(config); err != nil {
			return err
		}
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
		if err := processImage(image, config); err != nil {
			return fmt.Errorf("failed to process image %s: %w", image, err)
		}
	}

	return nil
}
