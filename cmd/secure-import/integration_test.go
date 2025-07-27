package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTest(t *testing.T) (string, string, func()) {
	ctx := context.Background()

	// Start a registry container
	req := testcontainers.ContainerRequest{
		Image:        "registry:2",
		ExposedPorts: []string{"5000/tcp"},
		WaitingFor:   wait.ForListeningPort("5000/tcp"),
	}
	registryContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
	})
	if err != nil {
		t.Fatalf("Failed to create container: %s", err)
	}
	if err := registryContainer.Start(ctx); err != nil {
		t.Fatalf("Failed to start container: %s", err)
	}

	// Get the container's host and port
	host, err := registryContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %s", err)
	}
	port, err := registryContainer.MappedPort(ctx, "5000")
	if err != nil {
		t.Fatalf("Failed to get container port: %s", err)
	}
	registryURL := fmt.Sprintf("%s:%s", host, port.Port())

	// Create dummy images and push them to the local registry
	dummyImageName1_0_0 := fmt.Sprintf("%s/test/dummy-image:1.0.0", registryURL)
	dummyImage1_0_0, err := crane.Image(map[string][]byte{"/": {}})
	if err != nil {
		t.Fatalf("Failed to create dummy image 1.0.0: %v", err)
	}
	if err := crane.Push(dummyImage1_0_0, dummyImageName1_0_0, crane.Insecure); err != nil {
		t.Fatalf("Failed to push dummy image 1.0.0 to local registry: %v", err)
	}

	dummyImageName1_22_0 := fmt.Sprintf("%s/test/dummy-image:1.22.0", registryURL)
	dummyImage1_22_0, err := crane.Image(map[string][]byte{"/": {}})
	if err != nil {
		t.Fatalf("Failed to create dummy image 1.22.0: %v", err)
	}
	if err := crane.Push(dummyImage1_22_0, dummyImageName1_22_0, crane.Insecure); err != nil {
		t.Fatalf("Failed to push dummy image 1.22.0 to local registry: %v", err)
	}

	// Create a dummy chart
	chartDir, err := os.MkdirTemp("", "helm-chart-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}

	chartYaml := `
apiVersion: v2
name: my-chart
version: 1.2.3
description: A Helm chart for Kubernetes
`
	valuesYaml := fmt.Sprintf(`
image:
  repository: %s
  tag: 1.0.0
`, fmt.Sprintf("%s/test/dummy-image", registryURL))

	if err := os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYaml), 0644); err != nil {
		t.Fatalf("Failed to write Chart.yaml: %s", err)
	}
	if err := os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(valuesYaml), 0644); err != nil {
		t.Fatalf("Failed to write values.yaml: %s", err)
	}

	// Create a local repository directory
	repoDir, err := os.MkdirTemp("", "helm-repo-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}

	// Package the chart into the repository directory
	if err := execCommand("helm", "package", chartDir, "--destination", repoDir); err != nil {
		t.Fatalf("Failed to package chart: %s", err)
	}

	// Index the repository
	if err := execCommand("helm", "repo", "index", repoDir); err != nil {
		t.Fatalf("Failed to index repo: %s", err)
	}

	// Start a local HTTP server to serve the repository
	repoServer := httptest.NewServer(http.FileServer(http.Dir(repoDir)))
	repoURL := repoServer.URL

	return registryURL, repoURL, func() {
		// Teardown
		if err := registryContainer.Terminate(ctx); err != nil {
			t.Fatalf("Failed to terminate container: %s", err)
		}
		repoServer.Close()
		os.RemoveAll(chartDir)
		os.RemoveAll(repoDir)
	}
}

func TestImportChart(t *testing.T) {
	registryURL, repoURL, teardown := setupTest(t)
	defer teardown()

	config := &Config{
		ChartName: "my-chart",
		Version:   "1.2.3",
		Repo:      repoURL,
		Registry:  registryURL,
		ChartFile: "my-chart-1.2.3.tgz",
		ReportFormat: "table",
	}

	err := run(config)
	assert.NoError(t, err)

	// Verify that the chart and image were pushed to the registry
	chartExists, err := imageExists(fmt.Sprintf("%s/charts/%s:%s", registryURL, config.ChartName, config.Version))
	assert.NoError(t, err)
	assert.True(t, chartExists, "Chart should exist in the registry")

	imageExists, err := imageExists(fmt.Sprintf("%s/test/dummy-image:1.0.0", registryURL))
	assert.NoError(t, err)
	assert.True(t, imageExists, "Image should exist in the registry")
}

func TestImportChartWithValues(t *testing.T) {
	registryURL, repoURL, teardown := setupTest(t)
	defer teardown()

	// Create a dummy values file
	valuesFile, err := os.CreateTemp("", "values-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %s", err)
	}
	defer os.Remove(valuesFile.Name())

	valuesYaml := `
image:
  tag: 1.22.0
`
	if _, err := valuesFile.WriteString(valuesYaml); err != nil {
		t.Fatalf("Failed to write to temp file: %s", err)
	}
	valuesFile.Close()

	config := &Config{
		ChartName: "my-chart",
		Version:   "1.2.3",
		Repo:      repoURL,
		Registry:  registryURL,
		Values:    valuesFile.Name(),
		ChartFile: "my-chart-1.2.3.tgz",
		ReportFormat: "table",
	}

	err = run(config)
	assert.NoError(t, err)

	// Verify that the chart and image were pushed to the registry
	chartExists, err := imageExists(fmt.Sprintf("%s/charts/%s:%s", registryURL, config.ChartName, config.Version))
	assert.NoError(t, err)
	assert.True(t, chartExists, "Chart should exist in the registry")

	// The dummy image tag is 1.0.0, but the values file overrides it to 1.22.0
	// So we expect the image to be dummy-image:1.22.0
	imageExists, err := imageExists(fmt.Sprintf("%s/test/dummy-image:1.22.0", registryURL))
	assert.NoError(t, err)
	assert.True(t, imageExists, "Image should exist in the registry")
}