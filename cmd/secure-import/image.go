package main

import (
	"net/http"
	"fmt"
	"path/filepath"
	"strings"
)

type TrivyResult struct {
	Results []struct {
		Vulnerabilities []interface{} `json:"Vulnerabilities"`
	} `json:"Results"`
}


func processImage(image string, config *Config) error {
	parts := strings.Split(image, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid image format: %s", image)
	}

	registry := parts[0]
	nameWithTag := strings.Join(parts[1:], "/")
	lastIndex := strings.LastIndex(nameWithTag, ":")
	if lastIndex == -1 {
		return fmt.Errorf("no tag found in image: %s", image)
	}

	name := nameWithTag[:lastIndex]
	tag := nameWithTag[lastIndex+1:]
	finalImage := fmt.Sprintf("%s/%s:%s", config.Registry, name, tag)

	// Check if image exists using registry API
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", 
		config.Registry, name, tag)
	
	resp, err := http.Head(url)
	exists := err == nil && resp.StatusCode == http.StatusOK
	if resp != nil {
		resp.Body.Close()
	}

	if !exists {
		if err := processNewImage(image, registry, name, tag, finalImage, config); err != nil {
			return err
		}
	} else {
		fmt.Printf("Image %s:%s already exists in registry. Skipping push.\n", name, tag)
	}

	return nil
}

func processNewImage(image, registry, name, tag, finalImage string, config *Config) error {
	// Pull image
	if err := execCommand("docker", "pull", image); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Run Trivy scan
	jsonFile := filepath.Base(image) + ".json"
	if err := execCommand("trivy", "image",
		"--vuln-type", "os",
		"--ignore-unfixed",
		"-f", "json",
		"-o", jsonFile,
		image); err != nil {
		return fmt.Errorf("failed to run Trivy scan: %w", err)
	}

	// Check vulnerabilities
	hasVulns, err := checkVulnerabilities(jsonFile)
	if err != nil {
		return err
	}

	if hasVulns {
		fmt.Printf("Patching %s:%s...\n", name, tag)
		if err := execCommand("copa", "patch",
			"-r", jsonFile,
			"-i", image,
			"-t", "patched"); err != nil {
			return fmt.Errorf("failed to patch image: %w", err)
		}
		if err := execCommand("docker", "tag",
			fmt.Sprintf("%s/%s:patched", registry, name),
			finalImage); err != nil {
			return fmt.Errorf("failed to tag patched image: %w", err)
		}
	} else {
		fmt.Println("No vulnerabilities were found.")
		if err := execCommand("docker", "tag", image, finalImage); err != nil {
			return fmt.Errorf("failed to tag image: %w", err)
		}
	}

	// Run post-patch Trivy scan
	if err := execCommand("trivy", "image",
		"--vuln-type", "os",
		"--ignore-unfixed",
		finalImage); err != nil {
		return fmt.Errorf("failed to run post-patch Trivy scan: %w", err)
	}

	// Push image
	if err := execCommand("docker", "push", finalImage); err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	// Get digest using registry API
	digest, err := getDigest(config.Registry, name, tag)
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}

	// Sign image (unchanged)
	return execCommand("cosign", "sign",
		"--tlog-upload=false",
		"--key", config.SignKey,
		fmt.Sprintf("%s/%s@%s",
			config.Registry, name, digest))
}
