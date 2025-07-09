package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
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
	fmt.Printf("  Checking if image exists: %s\n", imageRef)

	opts := []crane.Option{crane.WithAuthFromKeychain(authn.DefaultKeychain)}
	if strings.HasPrefix(imageRef, "localhost:") {
		opts = append(opts, crane.Insecure)
	}

	_, err := crane.Head(imageRef, opts...)
	if err != nil {
		// Check if the error is a transport error indicating the image was not found (404).
		if tErr, ok := err.(*transport.Error); ok {
			if tErr.StatusCode == 404 {
				fmt.Printf("  Image %s does not exist (404)\n", imageRef)
				return false, nil
			}
		}
		// For other errors (like authentication failures), return the error.
		fmt.Printf("  Error checking image existence for %s: %v\n", imageRef, err)
		return false, fmt.Errorf("error checking image existence: %w", err)
	}
	fmt.Printf("  Image %s exists.\n", imageRef)
	return true, nil
}

func getDigest(registry, repository, reference string) (string, error) {
	imageRef := fmt.Sprintf("%s/%s:%s", registry, repository, reference)
	fmt.Printf("  Getting digest for: %s\n", imageRef)

	opts := []crane.Option{crane.WithAuthFromKeychain(authn.DefaultKeychain)}
	if strings.HasPrefix(imageRef, "localhost:") {
		opts = append(opts, crane.Insecure)
	}

	// Get the digest using crane with keychain authentication.
	digest, err := crane.Digest(imageRef, opts...)
	if err != nil {
		return "", fmt.Errorf("getting digest: %w", err)
	}
	fmt.Printf("  Digest for %s: %s\n", imageRef, digest)
	return digest, nil
}

func processImage(image string, config *Config) error {
	fmt.Printf("Processing image: %s\n", image)
	parts := strings.Split(image, "/")

	var originalRegistry string
	var nameWithTag string

	if len(parts) == 1 || (!strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":")) {
		// Assume docker.io for images without a specified registry or with a single part (e.g., "nginx:latest")
		originalRegistry = "docker.io"
		nameWithTag = image
	} else {
		originalRegistry = parts[0]
		nameWithTag = strings.Join(parts[1:], "/")
	}

	lastIndexOf := strings.LastIndex(nameWithTag, ":")
	if lastIndexOf == -1 {
		return fmt.Errorf("no tag found in image: %s", image)
	}

	name := nameWithTag[:lastIndexOf]
	tag := nameWithTag[lastIndexOf+1:]
	finalImage := fmt.Sprintf("%s/%s:%s", config.Registry, name, tag)

	fmt.Printf("  Original Registry: %s\n", originalRegistry)
	fmt.Printf("  Name: %s\n", name)
	fmt.Printf("  Tag: %s\n", tag)
	fmt.Printf("  Final Image: %s\n", finalImage)

	exists, err := imageExists(finalImage)
	if err != nil {
		return err
	}

	if !exists {
		fmt.Printf("  Image %s does not exist in target registry. Processing new image.\n", finalImage)
		if err := processNewImage(image, originalRegistry, name, tag, finalImage, config); err != nil {
			return err
		}
	} else {
		fmt.Printf("  Image %s already exists in registry. Skipping push.\n", finalImage)
	}

	return nil
}

func processNewImage(image, registry, name, tag, finalImage string, config *Config) error {
	pullImageRef := fmt.Sprintf("%s/%s:%s", registry, name, tag)
	fmt.Printf("  Pulling image: %s\n", pullImageRef)

	opts := []crane.Option{crane.WithAuthFromKeychain(authn.DefaultKeychain)}
	if strings.HasPrefix(pullImageRef, "localhost:") {
		opts = append(opts, crane.Insecure)
	}

	img, err := crane.Pull(pullImageRef, opts...)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", pullImageRef, err)
	}
	fmt.Printf("  Successfully pulled image: %s\n", pullImageRef)

	if isInstalled("copa") {
		fmt.Println("  Copa is available. Attempting to patch image.")
		img, err = patchImage(name, tag, image, registry, img)
		if err != nil {
			return fmt.Errorf("failed to patch image: %w", err)
		}
		fmt.Println("  Image patched successfully.")
	} else {
		fmt.Println("  Skipping patching - copa is not available")
	}

	fmt.Printf("  Pushing image to final destination: %s\n", finalImage)
	opts = []crane.Option{crane.WithAuthFromKeychain(authn.DefaultKeychain)}
	if strings.HasPrefix(finalImage, "localhost:") {
		opts = append(opts, crane.Insecure)
	}

	if err := crane.Push(img, finalImage, opts...); err != nil {
		return fmt.Errorf("failed to push image %s: %w", finalImage, err)
	}
	fmt.Printf("  Successfully pushed image: %s\n", finalImage)

	if isInstalled("trivy") {
		fmt.Println("  Trivy is available. Running post-patch Trivy scan.")
		if err := execCommand("trivy", "image",
			"--vuln-type", "os",
			"--ignore-unfixed",
			finalImage); err != nil {
			return fmt.Errorf("failed to run post-patch Trivy scan: %w", err)
		}
		fmt.Println("  Post-patch Trivy scan completed.")
	} else {
		fmt.Println("  Skipping vulnerability scanning - trivy is not available")
	}

	if !isInstalled("cosign") {
		fmt.Println("  Skipping image signing - cosign is not available")
		return nil
	}

	if !config.Sign {
		fmt.Println("  Skipping image signing as no signing key was provided")
		return nil
	}

	digest, err := getDigest(config.Registry, name, tag)
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}

	fmt.Printf("  Signing image: %s/%s@%s\n", config.Registry, name, digest)
	return execCommand("cosign", "sign",
		"--tlog-upload=false",
		"--key", config.SignKey,
		fmt.Sprintf("%s/%s@%s",
			config.Registry, name, digest))
}

func patchImage(name, tag, image, registry string, img v1.Image) (v1.Image, error) {
	fmt.Printf("  Patching image: %s:%s\n", name, tag)
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

	hasVulns, err := checkVulnerabilities(jsonFile)
	if err != nil {
		return nil, err
	}

	if hasVulns {
		fmt.Printf("  Vulnerabilities found. Patching %s:%s...\n", name, tag)
		if err := execCommand("copa", "patch",
			"-r", jsonFile,
			"-i", image,
			"-t", "patched"); err != nil {
			return nil, fmt.Errorf("failed to patch image: %w", err)
		}

		ref, err := v1name.ParseReference(fmt.Sprintf("%s/%s:patched", registry, name))
		if err != nil {
			return nil, fmt.Errorf("failed to parse reference: %w", err)
		}

		img, err = daemon.Image(ref)
		if err != nil {
			return nil, fmt.Errorf("failed to get patched image from daemon: %w", err)
		}
	} else {
		fmt.Println("  No vulnerabilities were found.")
	}
	return img, nil
}