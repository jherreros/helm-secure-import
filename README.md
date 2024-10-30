# Helm Registry Import Plugin

A Helm plugin that imports charts and their container images into an OCI-compliant Registry, with built-in vulnerability scanning, patching, and signing capabilities.

## Features

- Pulls Helm charts from any repository
- Pushes charts to OCI-compliant Registry
- Signs charts using Cosign
- Extracts and processes container images from charts:
  - Scans for vulnerabilities using Trivy
  - Automatically patches vulnerable images using Copa
  - Pushes images to registry
  - Signs images using Cosign

## Prerequisites

- Helm v3.x
- Docker

## Installation

```bash
helm plugin install https://github.com/jherreros/helm-secure-import
```

## Usage

```bash
helm secure-import --chart <chart-name> \
                    --version <chart-version> \
                    --repo-url <repository-url> \
                    --registry <registry-url> \
                    [--sign-key <cosign-key-path>] \
                    [--values <values-file>]
```

Further examples of how to use the plugin can be found under [examples](examples/basic/basic.md).

### Parameters

- `--chart`: Name of the Helm chart
- `--version`: Version of the chart to import
- `--repo-url`: URL of the Helm repository
- `--registry`: Name of the OCI-compliant Registry
- `--sign-key` (optional): Path to the Cosign signing key or KMS URI
- `--values` (optional): Path to a Helm values file

### Example

```bash
helm secure-import --chart nginx \
                    --version 1.2.3 \
                    --repo-url https://charts.bitnami.com/bitnami \
                    --registry my.registry.io \
                    --sign-key /path/to/cosign.key \
                    --values ./values.yaml
```

## Security Features

### Vulnerability Scanning
The plugin uses Trivy to scan all container images for vulnerabilities before importing them. Only OS-level vulnerabilities with available fixes are considered.

### Image Patching
If vulnerabilities are found, the plugin automatically patches the images using Copa before pushing them to your registry.

### Signing
Both charts and container images are signed using Cosign before being pushed to your registry. This ensures the integrity and authenticity of the artifacts.

## Acknowledgements

The idea for this plugin is inspired in [Helmper](https://github.com/ChristofferNissen/helmper).

## License

This project is licensed under the MIT license.
