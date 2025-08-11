package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Version is the current version of the plugin
const Version = "0.3.2"

// Config holds the application configuration
// It's populated from command-line flags and environment variables.

type Config struct {
	ChartName    string
	Version      string
	Repo         string
	Values       string
	Registry     string
	SignKey      string
	ChartFile    string
	Sign         bool
	IsOCI        bool
	ReportFormat string
	ReportFile   string
	DryRun       bool
}

func parseFlags() (*Config, error) {
	// Handle standalone --version flag (not followed by a value)
	cmdArgs := os.Args[1:]
	for i, arg := range cmdArgs {
		if arg == "--version" {
			// Check if this is the standalone version command (not followed by a value or is the last arg)
			if i == len(cmdArgs)-1 || (i+1 < len(cmdArgs) && strings.HasPrefix(cmdArgs[i+1], "-")) {
				fmt.Printf("helm-secure-import version %s\n", Version)
				os.Exit(0)
			}
		} else if arg == "-version" {
			fmt.Printf("helm-secure-import version %s\n", Version)
			os.Exit(0)
		}
	}

	config := &Config{}

	flag.StringVar(&config.ChartName, "chart", "", "Chart name (required)")
	flag.StringVar(&config.Version, "version", "", "Chart version (required)")
	flag.StringVar(&config.Repo, "repo", "", "Repository URL (can be HTTP or OCI)")
	flag.StringVar(&config.Values, "values", "", "Values file (optional)")
	flag.StringVar(&config.Registry, "registry", "", "Destination registry URL (can also be set via HELM_REGISTRY env var)")
	flag.StringVar(&config.SignKey, "sign-key", "", "Signing key (optional)")
	flag.StringVar(&config.ReportFormat, "report-format", "table", "Report format (table or json)")
	flag.StringVar(&config.ReportFile, "report-file", "", "Report file (for json format)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be imported without actually doing it")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSecurely imports all images in a helm chart into a container registry.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  HELM_REGISTRY    Registry URL (alternative to --registry flag)\n")
		fmt.Fprintf(os.Stderr, "  HELM_SIGN_KEY    Signing key path (alternative to --sign-key flag)\n")
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

	if config.SignKey == "" {
		config.SignKey = os.Getenv("HELM_SIGN_KEY")
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
		return nil, fmt.Errorf("\nMissing required flags: %v. Run 'helm secure-import --help' for usage information", missingFlags)
	}

	// Validate chart name format (alphanumeric, hyphens, underscores)
	if !isValidChartName(config.ChartName) {
		return nil, fmt.Errorf("invalid chart name '%s': must contain only alphanumeric characters, hyphens, and underscores", config.ChartName)
	}

	// Validate semantic versioning format
	if !isValidVersion(config.Version) {
		return nil, fmt.Errorf("invalid version '%s': must follow semantic versioning format (e.g., 1.2.3, 1.0.0-alpha)", config.Version)
	}

	// Validate values file exists if provided
	if config.Values != "" {
		if _, err := os.Stat(config.Values); os.IsNotExist(err) {
			return nil, fmt.Errorf("values file does not exist: %s", config.Values)
		}
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

// isValidChartName validates that the chart name contains only valid characters
func isValidChartName(name string) bool {
	// Chart names should contain only alphanumeric characters, hyphens, and underscores
	// and should not be empty
	if name == "" {
		return false
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", name)
	return matched
}

// isValidVersion validates that the version follows semantic versioning format
func isValidVersion(version string) bool {
	// Basic semantic versioning pattern: major.minor.patch with optional pre-release and build metadata
	if version == "" {
		return false
	}
	// Allow basic semver patterns like 1.2.3, 1.0.0-alpha, 1.2.3-beta.1, etc.
	matched, _ := regexp.MatchString(`^v?(\d+)\.(\d+)\.(\d+)(-[a-zA-Z0-9\.-]+)?(\+[a-zA-Z0-9\.-]+)?$`, version)
	return matched
}
