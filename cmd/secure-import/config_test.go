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
		name          string
		args          []string
		env           map[string]string
		expectErr     bool
		expectedErrStr string
		expectedConfig *Config
	}{
		{
			name: "All flags provided",
			args: []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--sign-key", "my-key", "--values", "my-values.yaml"},
			env:  map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName: "my-chart",
				Version:   "1.2.3",
				Repo:      "my-repo",
				Registry:  "my.registry.io",
				SignKey:   "my-key",
				Values:    "my-values.yaml",
				ChartFile: "my-chart-1.2.3.tgz",
				Sign:      true,
				ReportFormat: "table",
			},
		},
		{
			name: "Missing required flags",
			args: []string{"cmd"},
			env:  map[string]string{},
			expectErr: true,
			expectedErrStr: "missing required flags: [chart version repo registry]",
		},
		{
			name: "Registry from env var",
			args: []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo"},
			env:  map[string]string{"HELM_REGISTRY": "my.registry.io"},
			expectErr: false,
			expectedConfig: &Config{
				ChartName: "my-chart",
				Version:   "1.2.3",
				Repo:      "my-repo",
				Registry:  "my.registry.io",
				ChartFile: "my-chart-1.2.3.tgz",
				Sign:      false,
				ReportFormat: "table",
			},
		},
		{
			name: "Invalid registry format",
			args: []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "invalid-registry"},
			env:  map[string]string{},
			expectErr: true,
			expectedErrStr: "invalid registry format: invalid-registry",
		},
		{
			name: "Positional argument for chart name",
			args: []string{"cmd", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io"},
			env:  map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName: "my-chart",
				Version:   "1.2.3",
				Repo:      "my-repo",
				Registry:  "my.registry.io",
				ChartFile: "my-chart-1.2.3.tgz",
				Sign:      false,
				ReportFormat: "table",
			},
		},
		{
			name: "JSON report",
			args: []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--report-format", "json", "--report-file", "report.json"},
			env:  map[string]string{},
			expectErr: false,
			expectedConfig: &Config{
				ChartName: "my-chart",
				Version:   "1.2.3",
				Repo:      "my-repo",
				Registry:  "my.registry.io",
				ChartFile: "my-chart-1.2.3.tgz",
				Sign:      false,
				ReportFormat: "json",
				ReportFile: "report.json",
			},
		},
		{
			name: "Invalid report format",
			args: []string{"cmd", "--chart", "my-chart", "--version", "1.2.3", "--repo", "my-repo", "--registry", "my.registry.io", "--report-format", "xml"},
			env:  map[string]string{},
			expectErr: true,
			expectedErrStr: "invalid report format: xml",
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