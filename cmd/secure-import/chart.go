package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func pushAndSignChart(config *Config) error {
	// Push chart using helm push (unchanged)
	if err := execCommand("helm", "push", config.ChartFile,
		fmt.Sprintf("oci://%s/charts/", config.Registry)); err != nil {
		return fmt.Errorf("failed to push chart: %w", err)
	}

	// Skip signing if no key provided
	if !config.Sign {
		fmt.Println("Skipping chart signing as no signing key was provided")
		return nil
	}

	// Get digest using registry API
	digest, err := getDigest(config.Registry, 
		fmt.Sprintf("charts/%s", config.ChartName), 
		config.Version)
	if err != nil {
		return fmt.Errorf("failed to get chart digest: %w", err)
	}

	// Sign chart (unchanged)
	return execCommand("cosign", "sign",
		"--tlog-upload=false",
		"--key", config.SignKey,
		fmt.Sprintf("%s/charts/%s@%s",
			config.Registry, config.ChartName, digest))
}

func getImagesFromChart(config *Config) ([]string, error) {
	args := []string{"template", config.ChartFile}
	if config.Values != "" {
		args = append(args, "-f", config.Values)
	}

	cmd := exec.Command("helm", args...)
	helmOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to template chart: %w", err)
	}

	// Use yq to extract images
	yqCmd := exec.Command("yq", "e", ".. | select(has(\"image\")) | .image", "-")
	yqCmd.Stdin = strings.NewReader(string(helmOutput))
	yqOutput, err := yqCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract images: %w", err)
	}

	// Define regex pattern for valid image references
	pattern := `^[a-zA-Z0-9][a-zA-Z0-9.-]*(:[0-9]+)?/[a-zA-Z0-9/_-]+(/[a-zA-Z0-9/_-]+)?:[a-zA-Z0-9._-]+$`
	regex := regexp.MustCompile(pattern)

	// Filter images
	var images []string
	for _, line := range strings.Split(string(yqOutput), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && regex.MatchString(line) {
			images = append(images, line)
		}
	}

	return images, nil
}
