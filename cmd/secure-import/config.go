package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ChartName string
	Version   string
	Repo   string
	Values    string
	Registry  string
	SignKey   string
	ChartFile string
	Sign      bool
}

func parseFlags() (*Config, error) {
	config := &Config{}
	
	flag.StringVar(&config.ChartName, "chart", "", "Chart name (required)")
	flag.StringVar(&config.Version, "version", "", "Chart version (required)")
	flag.StringVar(&config.Repo, "repo", "", "Repository (required)")
	flag.StringVar(&config.Values, "values", "", "Values file (optional)")
	flag.StringVar(&config.Registry, "registry", "", "Registry URL (can also be set via HELM_REGISTRY env var)")
	flag.StringVar(&config.SignKey, "sign-key", "", "Signing key (optional)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSecurely imports all images in a helm chart into a container registry.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  HELM_REGISTRY    Registry URL (alternative to --registry flag)\n")
	}

	flag.Parse()

	// Check for positional arguments
	args := flag.Args()
	if config.ChartName == "" && len(args) > 0 {
		// Use the first positional argument as the chart name
		config.ChartName = args[0]
		// Remove the chart name from the positional arguments to avoid confusion
		args = args[1:]
	}

	// Re-parse the remaining positional arguments as flags
	// This allows flags to appear after the chart name
	if len(args) > 0 {
		if err := flag.CommandLine.Parse(args); err != nil {
			return nil, fmt.Errorf("failed to parse additional flags: %v", err)
		}
	}

	// Check environment variable for registry if not set via flag
	if config.Registry == "" {
		config.Registry = os.Getenv("HELM_REGISTRY")
	}
	
	// Required flag validation
	var missingFlags []string
	
	if config.ChartName == "" {
		missingFlags = append(missingFlags, "chart")
	}
	if config.Version == "" {
		missingFlags = append(missingFlags, "version")
	}
	if config.Repo == "" {
		missingFlags = append(missingFlags, "repo")
	}
	if config.Registry == "" {
		missingFlags = append(missingFlags, "registry")
	}

	if len(missingFlags) > 0 {
		flag.Usage()
		return nil, fmt.Errorf("\nMissing required flags: %v", missingFlags)
	}

	if !strings.HasPrefix(config.Registry, "localhost:") && 
		!strings.Contains(config.Registry, ".") {
		return nil, fmt.Errorf("invalid registry format: %s", config.Registry)
	}	

	// Set derived fields
	config.ChartFile = fmt.Sprintf("%s-%s.tgz", config.ChartName, config.Version)
	config.Sign = config.SignKey != ""

	return config, nil
}
