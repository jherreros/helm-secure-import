package main

import (
	"os"
	"testing"
)

func TestIsInstalled(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a dummy executable file
	dummyExecutable := tempDir + "/dummy"
	if _, err := os.Create(dummyExecutable); err != nil {
		t.Fatalf("Failed to create dummy executable: %v", err)
	}
	if err := os.Chmod(dummyExecutable, 0755); err != nil {
		t.Fatalf("Failed to make dummy executable executable: %v", err)
	}

	// Add the temporary directory to the PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", tempDir+":"+originalPath)

	// Test if the dummy executable is found
	if !isInstalled("dummy") {
		t.Errorf("Expected isInstalled to return true for dummy executable, but got false")
	}

	// Test for a non-existent command
	if isInstalled("nonexistentcommand") {
		t.Errorf("Expected isInstalled to return false for nonexistentcommand, but got true")
	}
}

func TestCheckVulnerabilities(t *testing.T) {
	// Create a temporary file with some dummy Trivy JSON output
	tempFile, err := os.CreateTemp("", "trivy-output-*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Test case 1: No vulnerabilities
	noVulnerabilitiesJSON := `{"Results": [{"Vulnerabilities": []}]}`
	if _, err := tempFile.WriteString(noVulnerabilitiesJSON); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	hasVulnerabilities, err := checkVulnerabilities(tempFile.Name())
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if hasVulnerabilities {
		t.Errorf("Expected hasVulnerabilities to be false, but got true")
	}

	// Test case 2: With vulnerabilities
	tempFile, err = os.Create(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	vulnerabilitiesJSON := `{"Results": [{"Vulnerabilities": [{"Severity": "HIGH"}]}]}`
	if _, err := tempFile.WriteString(vulnerabilitiesJSON); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	hasVulnerabilities, err = checkVulnerabilities(tempFile.Name())
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if !hasVulnerabilities {
		t.Errorf("Expected hasVulnerabilities to be true, but got false")
	}
}
