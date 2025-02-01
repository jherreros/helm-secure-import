package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	v1name "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type TrivyResult struct {
	Results []struct {
		Vulnerabilities []interface{} `json:"Vulnerabilities"`
	} `json:"Results"`
}

func imageExists(imageRef string) (bool, error) {
	// Attempt to get the manifest
	_, err := crane.Manifest(imageRef)
	if err != nil {
		// Check if the error is specifically "not found"
		if remoteErr, ok := err.(*transport.Error); ok {
			if remoteErr.StatusCode == 404 {
				return false, nil
			}
		}
		// For other errors, return the error to handle upstream
		return false, fmt.Errorf("error checking image: %w", err)
	}

	return true, nil
}

func getDigest(registry, repository, reference string) (string, error) {
	imageRef := fmt.Sprintf("%s/%s:%s", registry, repository, reference)

	// Get the digest using crane
	digest, err := crane.Digest(imageRef)
	if err != nil {
		return "", fmt.Errorf("getting digest: %w", err)
	}

	return digest, nil
}

func processImage(image string, config *Config) error {
	parts := strings.Split(image, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid image format: %s", image)
	}

	registry := parts[0]
	nameWithTag := strings.Join(parts[1:], "/")
	lastIndex := strings.LastIndex(nameWithTag, ":")
	if lastIndex == -1 {
		return fmt.Errorf("no tag found in image: %s", image)
	}

	name := nameWithTag[:lastIndex]
	tag := nameWithTag[lastIndex+1:]
	finalImage := fmt.Sprintf("%s/%s:%s", config.Registry, name, tag)
	
	exists, err := imageExists(finalImage)
	if err != nil {
		return err
	}

	if !exists {
		if err := processNewImage(image, registry, name, tag, finalImage, config); err != nil {
			return err
		}
	} else {
		fmt.Printf("Image %s:%s already exists in registry. Skipping push.\n", name, tag)
	}

	return nil
}

func processNewImage(image, registry, name, tag, finalImage string, config *Config) error {
	img, err := crane.Pull(image)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	if isInstalled("copa"){
		img, err = patchImage(name, tag, image, registry, img)
		if err != nil {
			return fmt.Errorf("failed to patch image: %w", err)
		}
	} else {
		fmt.Println("Skipping patching - copa is not available")
	}

	// Push the final image using crane
	if err := crane.Push(img, finalImage); err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	if isInstalled("trivy"){
		// Run post-patch Trivy scan
		if err := execCommand("trivy", "image",
			"--vuln-type", "os",
			"--ignore-unfixed",
			finalImage); err != nil {
			return fmt.Errorf("failed to run post-patch Trivy scan: %w", err)
		}
	} else {
		fmt.Println("Skipping vulnerability scanning - trivy is not available")
	}

	if !isInstalled("cosign") {
		fmt.Println("Skipping image signing - cosign is not available")
		return nil
	}

	if !config.Sign {
		fmt.Println("Skipping image signing as no signing key was provided")
		return nil
	}

	digest, err := getDigest(config.Registry, name, tag)
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}

	return execCommand("cosign", "sign",
		"--tlog-upload=false",
		"--key", config.SignKey,
		fmt.Sprintf("%s/%s@%s",
			config.Registry, name, digest))
}

func patchImage(name, tag, image, registry string, img v1.Image) (v1.Image, error) {
		// Use temp directory for Trivy scan results
		jsonFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s.json", 
			strings.ReplaceAll(name, "/", "-"), tag))
		defer os.Remove(jsonFile)

		if err := execCommand("trivy", "image",
			"--vuln-type", "os",
			"--ignore-unfixed",
			"-f", "json",
			"-o", jsonFile,
			image); err != nil {
			return nil, fmt.Errorf("failed to run Trivy scan: %w", err)
		}

		// Check vulnerabilities
		hasVulns, err := checkVulnerabilities(jsonFile)
		if err != nil {
			return nil, err
		}

		if hasVulns {
			fmt.Printf("Patching %s:%s...\n", name, tag)
			if err := execCommand("copa", "patch",
				"-r", jsonFile,
				"-i", image,
				"-t", "patched"); err != nil {
				return nil, fmt.Errorf("failed to patch image: %w", err)
			}
			
			// Get the patched image from local daemon
			ref, err := v1name.ParseReference(fmt.Sprintf("%s/%s:patched", registry, name))
			if err != nil {
				return nil, fmt.Errorf("failed to parse reference: %w", err)
			}
			
			img, err = daemon.Image(ref)
			if err != nil {
				return nil, fmt.Errorf("failed to get patched image from daemon: %w", err)
			}
		} else {
			fmt.Println("No vulnerabilities were found.")
		}
	return img, nil
}
