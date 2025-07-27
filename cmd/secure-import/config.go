package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds the application configuration
// It's populated from command-line flags and environment variables.

type Config struct {
	ChartName      string
	Version        string
	Repo           string
	Values         string
	Registry       string
	SignKey        string
	ChartFile      string
	Sign           bool
	IsOCI          bool
	ReportFormat   string
	ReportFile     string
}

func parseFlags() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.ChartName, "chart", "", "Chart name (required)")
	flag.StringVar(&config.Version, "version", "", "Chart version (required)")
	flag.StringVar(&config.Repo, "repo", "", "Repository URL (can be HTTP or OCI)")
	flag.StringVar(&config.Values, "values", "", "Values file (optional)")
	flag.StringVar(&config.Registry, "registry", "", "Destination registry URL (can also be set via HELM_REGISTRY env var)")
	flag.StringVar(&config.SignKey, "sign-key", "", "Signing key (optional)")
	flag.StringVar(&config.ReportFormat, "report-format", "table", "Report format (table or json)")
	flag.StringVar(&config.ReportFile, "report-file", "", "Report file (for json format)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSecurely imports all images in a helm chart into a container registry.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  HELM_REGISTRY    Registry URL (alternative to --registry flag)\n")
	}

	flag.Parse()

	args := flag.Args()
	if config.ChartName == "" && len(args) > 0 {
		config.ChartName = args[0]
		args = args[1:]
	}

	if len(args) > 0 {
		if err := flag.CommandLine.Parse(args); err != nil {
			return nil, fmt.Errorf("failed to parse additional flags: %v", err)
		}
	}

	if config.Registry == "" {
		config.Registry = os.Getenv("HELM_REGISTRY")
	}

	config.IsOCI = strings.HasPrefix(config.Repo, "oci://")

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
		return nil, fmt.Errorf("\nmissing required flags: %v", missingFlags)
	}

	if !strings.HasPrefix(config.Registry, "localhost:") &&
		!strings.Contains(config.Registry, ".") {
		return nil, fmt.Errorf("invalid registry format: %s", config.Registry)
	}

	if config.ReportFormat != "table" && config.ReportFormat != "json" {
		return nil, fmt.Errorf("invalid report format: %s", config.ReportFormat)
	}

	config.ChartFile = fmt.Sprintf("%s-%s.tgz", config.ChartName, config.Version)
	config.Sign = config.SignKey != ""

	return config, nil
}

