# Helm Registry Import Plugin

A Helm plugin that imports charts and their container images into an OCI-compliant Registry, with built-in vulnerability scanning, patching, and signing capabilities.

## Features

- Pulls Helm charts from any repository (including OCI)
- Pushes charts to OCI-compliant Registry
- Signs charts using Cosign
- Extracts and processes container images from charts:
  - Scans for vulnerabilities using Trivy
  - Automatically patches vulnerable images using Copa
  - Pushes images to registry
  - Signs images using Cosign
- Generates a report of the import process (table or JSON)

## Prerequisites

- Helm v3.x

### Prerequisites for specific functionalities

- Copa - For image patching
- Trivy - For image scanning and patching
- Cosign - For image signing

## Installation

```bash
helm plugin install https://github.com/jherreros/helm-secure-import
```

## Usage

```bash
helm secure-import --chart <chart-name> \
                    --version <chart-version> \
                    --repo <repository-url> \
                    --registry <registry-url> \
                    [--sign-key <cosign-key-path>] \
                    [--values <values-file>] \
                    [--report-format <format>] \
                    [--report-file <file>]
```

Further examples of how to use the plugin can be found under [examples](examples/basic/basic.md).

### Parameters

- `--chart`: Name of the Helm chart. The name of the chart can also be specified directly (e.g. `helm secure-import my-chart`)
- `--version`: Version of the chart to import
- `--repo`: URL of the Helm repository (can be HTTP or OCI)
- `--registry`: Name of the OCI-compliant Registry. Can also be set as an environment variable (`HELM_REGISTRY`).
- `--sign-key` (optional): Path to the Cosign signing key or KMS URI
- `--values` (optional): Path to a Helm values file
- `--report-format` (optional): Report format (table or json). Defaults to table.
- `--report-file` (optional): Report file path (for json format).

### Example

```bash
# From a traditional repository
helm secure-import nginx \
                    --version 1.2.3 \
                    --repo https://charts.bitnami.com/bitnami \
                    --registry my.registry.io \
                    --sign-key /path/to/cosign.key \
                    --values ./values.yaml

# From an OCI registry
helm secure-import my-chart \
                    --version 4.5.6 \
                    --repo oci://my.registry.io/charts \
                    --registry my.registry.io

# With JSON report
helm secure-import my-chart \
                    --version 4.5.6 \
                    --repo oci://my.registry.io/charts \
                    --registry my.registry.io \
                    --report-format json \
                    --report-file report.json
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


