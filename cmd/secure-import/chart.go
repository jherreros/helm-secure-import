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
	// Strict image reference pattern (anchored) used for validation
	anchoredPattern := `^([a-zA-Z0-9][a-zA-Z0-9.-]*(?::[0-9]+)?/)?[a-zA-Z0-9/_-]+(?:/[a-zA-Z0-9/_-]+)*:[a-zA-Z0-9._+-]+$`
	anchoredRe := regexp.MustCompile(anchoredPattern)

	decoder := yaml.NewDecoder(bytes.NewReader(yamlInput))
	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
		extractImagesFromNode(&node, anchoredRe, nil, &rawImages)
	}

	uniqueImages := make(map[string]bool)
	var result []string
	for _, img := range rawImages {
		if !anchoredRe.MatchString(img) {
			continue
		}
		if isLikelyPortReference(img) || isLikelyLabelNotImage(img) || isLikelyMetricOrRecordingRule(img) {
			continue
		}
		if !uniqueImages[img] {
			uniqueImages[img] = true
			result = append(result, img)
		}
	}
	sort.Strings(result)
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

// isLikelyMetricOrRecordingRule filters out Prometheus recording rule names that resemble image references.
// Typical patterns seen:
//
//	apiserver_request:availability30d
//	apiserver_request:burnrate1h / burnrate5m / burnrate30m / burnrate2h / burnrate6h / burnrate1d / burnrate3d
//	count:up0 / count:up1
//	node_namespace_pod_container:container_memory_cache / ..._rss / ..._swap / ..._working_set_bytes
//
// These should be excluded because they are metrics, not images.
func isLikelyMetricOrRecordingRule(img string) bool {
	lastColon := strings.LastIndex(img, ":")
	if lastColon == -1 {
		return false
	}
	repo := img[:lastColon]
	tag := img[lastColon+1:]

	// Quick allow if repo contains a slash (real image path/registry)
	if strings.Contains(repo, "/") {
		return false
	}
	// Repository candidates typical for metrics
	metricRepos := map[string]struct{}{
		"apiserver_request":            {},
		"count":                        {},
		"node_namespace_pod_container": {},
	}
	if _, ok := metricRepos[repo]; !ok {
		return false
	}

	lowerTag := strings.ToLower(tag)

	// Specific known prefixes
	if strings.HasPrefix(lowerTag, "availability") {
		return true
	}
	if strings.HasPrefix(lowerTag, "burnrate") {
		return true
	}
	if strings.HasPrefix(lowerTag, "container_memory_") || lowerTag == "container_memory_cache" || lowerTag == "container_memory_rss" || lowerTag == "container_memory_swap" {
		return true
	}
	if strings.HasPrefix(lowerTag, "up") { // e.g., up0, up1
		rest := lowerTag[2:]
		if rest != "" {
			allDigits := true
			for _, r := range rest {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return true
			}
		}
	}
	if len(lowerTag) > 2 {
		unit := lowerTag[len(lowerTag)-1]
		if unit == 's' || unit == 'm' || unit == 'h' || unit == 'd' {
			hasDigit := false
			for _, r := range lowerTag[:len(lowerTag)-1] {
				if r >= '0' && r <= '9' {
					hasDigit = true
					break
				}
			}
			if hasDigit {
				return true
			}
		}
	}
	return false
}

// extractImagesFromNode recursively searches for image strings in a YAML node.
func extractImagesFromNode(n *yaml.Node, anchoredRe, _ *regexp.Regexp, images *[]string) {
	// Mapping node: look for repository/tag pair first.
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
				*images = append(*images, fmt.Sprintf("%s:%s", repo, tag))
				return // don't recurse this map
			}
		}
	}

	// Scalar node: may be a direct image or contain one or more images inside.
	if n.Kind == yaml.ScalarNode {
		val := strings.TrimSpace(n.Value)
		if val == "" {
			return
		}
		if anchoredRe.MatchString(val) { // whole scalar is an image
			*images = append(*images, val)
			return
		}
		// Embedded flag pattern: something=IMAGE. Only consider last '=' segment.
		if strings.Contains(val, "=") {
			candidate := val[strings.LastIndex(val, "=")+1:]
			candidate = strings.TrimSpace(candidate)
			// Avoid picking up simple words; require at least one slash (image path) and anchored match.
			if strings.Contains(candidate, "/") && anchoredRe.MatchString(candidate) {
				*images = append(*images, candidate)
			}
		}
		return
	}

	// Recurse into children (documents / sequences / remaining maps).
	for _, child := range n.Content {
		extractImagesFromNode(child, anchoredRe, nil, images)
	}
}
