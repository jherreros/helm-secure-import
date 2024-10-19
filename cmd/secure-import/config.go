package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	ChartName string
	Version   string
	RepoURL   string
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
	flag.StringVar(&config.RepoURL, "repo-url", "", "Repository URL (required)")
	flag.StringVar(&config.Values, "values", "", "Values file (optional)")
	flag.StringVar(&config.Registry, "registry", "", "Registry URL (required)")
	flag.StringVar(&config.SignKey, "sign-key", "", "Signing key (optional)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSecurely imports all images in a helm chart into a container registry.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Required flag validation
	var missingFlags []string
	
	if config.ChartName == "" {
		missingFlags = append(missingFlags, "chart")
	}
	if config.Version == "" {
		missingFlags = append(missingFlags, "version")
	}
	if config.RepoURL == "" {
		missingFlags = append(missingFlags, "repo-url")
	}
	if config.Registry == "" {
		missingFlags = append(missingFlags, "registry")
	}

	if len(missingFlags) > 0 {
		flag.Usage()
		return nil, fmt.Errorf("\nMissing required flags: %v", missingFlags)
	}

	// Set derived fields
	config.ChartFile = fmt.Sprintf("%s-%s.tgz", config.ChartName, config.Version)
	config.Sign = config.SignKey != ""

	return config, nil
}
