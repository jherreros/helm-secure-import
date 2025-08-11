package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

func pushAndSignChart(config *Config) error {
	chartRef := fmt.Sprintf("%s/charts/%s:%s", config.Registry, config.ChartName, config.Version)

	if err := execCommand("helm", "push", config.ChartFile,
		fmt.Sprintf("oci://%s/charts/", config.Registry)); err != nil {
		return fmt.Errorf("failed to push chart: %w", err)
	}

	// Invalidate cache for the chart since we just pushed it
	invalidateCacheEntry(chartRef)

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
	// Pattern supports common container image references. Intentionally broad; we'll post-filter false positives.
	pattern := `^([a-zA-Z0-9][a-zA-Z0-9.-]*(?::[0-9]+)?/)?[a-zA-Z0-9/_-]+(?:/[a-zA-Z0-9/_-]+)*:[a-zA-Z0-9._+-]+$`
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

	// Deduplicate, validate, and filter out obvious host:port style entries mistakenly classified as images.
	uniqueImages := make(map[string]bool)
	var result []string
	for _, img := range rawImages {
		if !re.MatchString(img) { // Not an image candidate
			continue
		}
		if isLikelyPortReference(img) { // Exclude service:port style
			continue
		}
		if isLikelyLabelNotImage(img) { // Exclude label-like entries (e.g., crossplane:aggregate-to-admin)
			continue
		}
		if !uniqueImages[img] {
			uniqueImages[img] = true
			result = append(result, img)
		}
	}
	sort.Strings(result) // Deterministic output
	return result, nil
}

// isLikelyPortReference attempts to distinguish false positives like service-name:8080 from real images.
// Heuristics (kept conservative to avoid excluding legitimate images):
//   - No slash in the repository part (official library images also match this; keep them unless other conditions hit)
//   - Tag is only digits
//   - Tag length >= 4 (reduces impact on common numeric version tags like :1, :12, :123)
//   - Repository segment contains at least one hyphen (most official single-word library images like redis, nginx lack hyphens)
//
// This will filter things like release-name-argocd-repo-server:8081 or release-name-redis-ha-haproxy:6379.
// NOTE: This may still allow rare false positives; adjust heuristics as needed with more real-world data.
func isLikelyPortReference(img string) bool {
	lastColon := strings.LastIndex(img, ":")
	if lastColon == -1 {
		return false
	}
	repo := img[:lastColon]
	tag := img[lastColon+1:]

	if strings.Contains(repo, "/") { // Has a path component; treat as image
		return false
	}
	// Tag must be all digits
	for _, r := range tag {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	if len(tag) < 4 { // Allow short numeric tags like :1 or :13
		return false
	}
	// Repository must have a hyphen to differentiate from typical library images (redis, busybox, etc.)
	if !strings.Contains(repo, "-") {
		return false
	}
	return true
}

// isLikelyLabelNotImage filters out scalars that syntactically resemble an image but are likely RBAC/label keys or permission strings.
// Conditions:
//   - Single segment repository (no slash, no registry dot/port)
//   - Tag contains no digits
//   - Tag NOT in allowlist of common all-alpha tags used for real images
//
// Examples filtered: crossplane:aggregate-to-admin, crossplane:allowed-provider-permissions, crossplane:masters
// Kept: crossplane:v1.20.1, nginx:latest, alpine:latest (slash or allowed tag or digits present)
func isLikelyLabelNotImage(img string) bool {
	lastColon := strings.LastIndex(img, ":")
	if lastColon == -1 {
		return false
	}
	repo := img[:lastColon]
	tag := img[lastColon+1:]

	if strings.Contains(repo, "/") { // path component -> likely image
		return false
	}
	if strings.Contains(repo, ".") { // registry host -> image
		return false
	}
	if strings.Contains(repo, ":") { // port in repo -> image
		return false
	}
	hasDigit := false
	for _, r := range tag {
		if unicode.IsDigit(r) {
			hasDigit = true
			break
		}
	}
	if hasDigit { // version-like
		return false
	}
	allowed := map[string]struct{}{
		"latest": {}, "stable": {}, "dev": {}, "prod": {}, "test": {}, "canary": {},
		"alpine": {}, "scratch": {}, "distroless": {}, "slim": {},
	}
	if _, ok := allowed[tag]; ok {
		return false
	}
	// Tags composed solely of letters and hyphens are suspect if above conditions met
	for _, r := range tag {
		if !(unicode.IsLetter(r) || r == '-') {
			return false // contains other chars -> keep
		}
	}
	return true
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
