package main

import (
	"net/http"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func checkChartExists(config *Config) (bool, error) {	
	// Use Docker Registry HTTP API V2 to check if manifest exists
	url := fmt.Sprintf("https://%s/v2/charts/%s/manifests/%s", 
		config.Registry, config.ChartName, config.Version)
	
	resp, err := http.Head(url)
	if err != nil {
		return false, nil // Assume doesn't exist if request fails
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK, nil
}


// Get digest for an image or chart
func getDigest(registry, repository, reference string) (string, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", 
		registry, repository, reference)
	
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Accept Docker manifest v2
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get manifest: %w", err)
	}
	defer resp.Body.Close()
	
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("no digest found")
	}
	
	return digest, nil
}

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
