package main

import (
	"fmt"
	"io"
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"gopkg.in/yaml.v3"
)

func pushAndSignChart(config *Config) error {
	if err := execCommand("helm", "push", config.ChartFile,
		fmt.Sprintf("oci://%s/charts/", config.Registry)); err != nil {
		return fmt.Errorf("failed to push chart: %w", err)
	}

	if !isInstalled("cosign") {
		fmt.Println("Skipping image signing - cosign is not available")
		return nil
	}

	// Skip signing if no key provided
	if !config.Sign {
		fmt.Println("Skipping chart signing as no signing key was provided")
		return nil
	}

	digest, err := getDigest(config.Registry, 
		fmt.Sprintf("charts/%s", config.ChartName), 
		config.Version)
	if err != nil {
		return fmt.Errorf("failed to get chart digest: %w", err)
	}

	return execCommand("cosign", "sign",
		"--tlog-upload=false",
		"--key", config.SignKey,
		fmt.Sprintf("%s/charts/%s@%s",
			config.Registry, config.ChartName, digest))
}

func getImagesFromChart(config *Config) ([]string, error) {
	args := []string{"template", config.ChartFile}
	if config.Values != "" {
		if _, err := os.Stat(config.Values); err != nil {
			return nil, fmt.Errorf("values file not found: %s", config.Values)
		}
		args = append(args, "-f", config.Values)
	}

	cmd := exec.Command("helm", args...)
	helmOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to template chart: %w", err)
	}
	
	images, err := extractImages(helmOutput)
	if err != nil {
		return nil, fmt.Errorf("error extracting images: %v", err)
	}

	return images, nil
}

// extractImages takes a YAML input and returns a slice of valid image strings.
func extractImages(yamlInput []byte) ([]string, error) {
	var images []string

	// Define regex pattern for valid image references.
	// Adjust the pattern as needed.
	pattern := `^[a-zA-Z0-9][a-zA-Z0-9.-]*(?::[0-9]+)?/[a-zA-Z0-9/_-]+(?:/[a-zA-Z0-9/_-]+)?:[a-zA-Z0-9._-]+$`
	re := regexp.MustCompile(pattern)

	decoder := yaml.NewDecoder(bytes.NewReader(yamlInput))
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
		// Recursively search for "image" keys in the document.
		extractImagesFromNode(&node, re, &images)
	}

	return images, nil
}

// extractImagesFromNode recursively searches for nodes with the key "image"
// and, if found, validates the corresponding value using the regex.
func extractImagesFromNode(n *yaml.Node, re *regexp.Regexp, images *[]string) {
	switch n.Kind {
	case yaml.DocumentNode:
		// Document nodes typically have a single child.
		for _, child := range n.Content {
			extractImagesFromNode(child, re, images)
		}
	case yaml.MappingNode:
		// In a mapping, keys and values are stored in pairs.
		for i := 0; i < len(n.Content); i += 2 {
			keyNode := n.Content[i]
			valueNode := n.Content[i+1]

			// If the key is "image" and the value is a scalar, check the value.
			if keyNode.Value == "image" && valueNode.Kind == yaml.ScalarNode {
				trimmed := strings.TrimSpace(valueNode.Value)
				if trimmed != "" && re.MatchString(trimmed) {
					*images = append(*images, trimmed)
				}
			}
			// Recurse into both the key and value nodes.
			extractImagesFromNode(keyNode, re, images)
			extractImagesFromNode(valueNode, re, images)
		}
	case yaml.SequenceNode:
		// For sequences, iterate over each child.
		for _, child := range n.Content {
			extractImagesFromNode(child, re, images)
		}
	}
}
