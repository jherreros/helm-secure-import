package main

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
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
	var rawImages []string
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
		extractImagesFromNode(&node, re, &rawImages)
	}

	// Deduplicate and validate the results
	uniqueImages := make(map[string]bool)
	var result []string
	for _, img := range rawImages {
		if re.MatchString(img) && !uniqueImages[img] {
			uniqueImages[img] = true
			result = append(result, img)
		}
	}
	sort.Strings(result) // Sort for deterministic output
	return result, nil
}

// extractImagesFromNode recursively searches for image strings in a YAML node.
func extractImagesFromNode(n *yaml.Node, re *regexp.Regexp, images *[]string) {
	// Rule 1: Check for the `repository` and `tag` pattern in a map.
	if n.Kind == yaml.MappingNode {
		contentMap := make(map[string]string)
		for i := 0; i < len(n.Content); i += 2 {
			keyNode := n.Content[i]
			valueNode := n.Content[i+1]
			if valueNode.Kind == yaml.ScalarNode {
				contentMap[keyNode.Value] = valueNode.Value
			}
		}

		if repo, ok := contentMap["repository"]; ok {
			if tag, ok := contentMap["tag"]; ok {
				imageStr := fmt.Sprintf("%s:%s", repo, tag)
				*images = append(*images, imageStr)
				// We found the image structure, so we DON'T recurse further into this map's children.
				return
			}
		}
	}

	// Rule 2: If it's not an image structure map, check for scalar values that are full image strings.
	if n.Kind == yaml.ScalarNode {
		*images = append(*images, strings.TrimSpace(n.Value))
		return // Scalars have no children
	}

	// Rule 3: Recurse into children for documents, sequences, and maps that were not identified as image structures.
	for _, child := range n.Content {
		extractImagesFromNode(child, re, images)
	}
}