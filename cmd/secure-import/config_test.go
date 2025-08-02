package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFlags(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Save original environment
	originalEnv := os.Getenv("HELM_REGISTRY")
	defer os.Setenv("HELM_REGISTRY", originalEnv)

	testCases := []struct {
		name           string
		args           []string
		env            map[string]string
		expectErr      bool
		expectedErrStr string
		expectedConfig *Config
	}{
		{
			name:      "All flags provided",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--sign-key", "my-key", "--values", "test-values.yaml"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				SignKey:      "my-key",
				Values:       "test-values.yaml",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         true,
				ReportFormat: "table",
			},
		},
		{
			name:           "Missing required flags",
			args:           []string{"cmd"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "Missing required flags: [chart version repo registry]. Run 'helm secure-import --help' for usage information",
		},
		{
			name:      "Registry from env var",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo"},
			env:       map[string]string{"HELM_REGISTRY": "my.registry.io"},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         false,
				ReportFormat: "table",
			},
		},
		{
			name:           "Invalid registry format",
			args:           []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "invalid-registry"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid registry format: invalid-registry",
		},
		{
			name:      "Positional argument for chart name",
			args:      []string{"cmd", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         false,
				ReportFormat: "table",
			},
		},
		{
			name:      "JSON report",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--report-format", "json", "--report-file", "report.json"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         false,
				ReportFormat: "json",
				ReportFile:   "report.json",
			},
		},
		{
			name:           "Invalid report format",
			args:           []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--report-format", "xml"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid report format: xml",
		},
		{
			name:      "HELM_SIGN_KEY environment variable",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:       map[string]string{"HELM_SIGN_KEY": "/path/to/key"},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				SignKey:      "/path/to/key",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         true,
				ReportFormat: "table",
			},
		},
		{
			name:           "Invalid chart name",
			args:           []string{"cmd", "--chart", "invalid@chart!", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid chart name 'invalid@chart!': must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:           "Invalid version format",
			args:           []string{"cmd", "--chart", "my-chart", "--version", "invalid-version!", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid version 'invalid-version!': must follow semantic versioning format",
		},
		{
			name:           "Non-existent values file",
			args:           []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--values", "non-existent.yaml"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "values file does not exist: non-existent.yaml",
		},
		{
			name:           "Empty chart name",
			args:           []string{"cmd", "--chart", "", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "Missing required flags: [chart]",
		},
		{
			name:           "Chart name with special characters",
			args:           []string{"cmd", "--chart", "my-chart@special!", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid chart name 'my-chart@special!': must contain only alphanumeric characters",
		},
		{
			name:           "Chart name with spaces",
			args:           []string{"cmd", "--chart", "my chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid chart name 'my chart': must contain only alphanumeric characters",
		},
		{
			name:           "Version with only major.minor",
			args:           []string{"cmd", "--chart", "my-chart", "--version", "1.2", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid version '1.2': must follow semantic versioning format",
		},
		{
			name:           "Version with invalid pre-release",
			args:           []string{"cmd", "--chart", "my-chart", "--version", "1.2.3-", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:            map[string]string{},
			expectErr:      true,
			expectedErrStr: "invalid version '1.2.3-': must follow semantic versioning format",
		},
		{
			name:      "Valid v-prefixed version",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "v1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "v1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				ChartFile:    "my-chart-v1.2.3.tgz",
				Sign:         false,
				ReportFormat: "table",
			},
		},
		{
			name:      "Valid localhost registry",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "localhost:5000"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "localhost:5000",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         false,
				ReportFormat: "table",
			},
		},
		{
			name:      "OCI repository",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "oci://registry.example.com/charts", "--registry", "my.registry.io"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "oci://registry.example.com/charts",
				Registry:     "my.registry.io",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         false,
				IsOCI:        true,
				ReportFormat: "table",
			},
		},
		{
			name:      "Dry run mode",
			args:      []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--dry-run"},
			env:       map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName:    "my-chart",
				Version:      "1.2.3",
				Repo:         "my-repo",
				Registry:     "my.registry.io",
				ChartFile:    "my-chart-1.2.3.tgz",
				Sign:         false,
				DryRun:       true,
				ReportFormat: "table",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flags for each test case
			flag.CommandLine = flag.NewFlagSet(tc.name, flag.ExitOnError)
			os.Args = tc.args

			// Set environment variables
			for k, v := range tc.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			config, err := parseFlags()

			if tc.expectErr {
				assert.Error(t, err)
				if tc.expectedErrStr != "" {
					assert.Contains(t, err.Error(), tc.expectedErrStr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedConfig, config)
			}
		})
	}
}

func TestIsValidChartName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid simple name", "nginx", true},
		{"Valid name with hyphens", "my-chart", true},
		{"Valid name with underscores", "my_chart", true},
		{"Valid alphanumeric", "chart123", true},
		{"Invalid with special chars", "chart@name", false},
		{"Invalid with spaces", "my chart", false},
		{"Invalid with dots", "my.chart", false},
		{"Empty string", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidChartName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid basic semver", "1.2.3", true},
		{"Valid with v prefix", "v1.2.3", true},
		{"Valid with pre-release", "1.2.3-alpha", true},
		{"Valid with build metadata", "1.2.3+build.1", true},
		{"Valid complex", "1.2.3-beta.1+build.2", true},
		{"Invalid format", "1.2", false},
		{"Invalid characters", "1.2.3!", false},
		{"Invalid pre-release", "1.2.3-", false},
		{"Empty string", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidVersion(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
