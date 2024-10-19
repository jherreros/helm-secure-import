# Basic Usage Example

This example demonstrates how to import the NGINX chart from the Bitnami repository to your OCI-compliant Registry.

## Prerequisites

1. OCI-compliant Registry access
2. Cosign key pair (if you don't have one, create it with `cosign generate-key-pair`)

## Command

Import the NGINX chart:
   ```bash
   helm secure-import \
     --chart nginx \
     --version 15.0.2 \
     --repo-url https://charts.bitnami.com/bitnami \
     --registry my.registry.io \
     --sign-key ./cosign.key
   ```

## Expected Output

```
Pulling chart nginx:15.0.2...
Chart downloaded successfully
Pushing chart to my.registry.io/charts/nginx:15.0.2...
Chart pushed successfully
Signing chart...
Chart signed successfully
Extracting images from chart...
Found images:
- docker.io/bitnami/nginx:1.24.0-debian-11-r0
Processing images...
Scanning for vulnerabilities...
No vulnerabilities found in nginx:1.24.0-debian-11-r0
Pushing image to my.registry.io/bitnami/nginx:1.24.0-debian-11-r0...
Image pushed successfully
Signing image...
Image signed successfully
```

The chart and its images are now available in your registry:
- Chart: `my.registry.io/charts/nginx:15.0.2`
- Image: `my.registry.io/bitnami/nginx:1.24.0-debian-11-r0`