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

### Getting Help

```bash
# Show usage information
helm secure-import --help

# Show version information  
helm secure-import --version
```

The `--help` flag shows all available options:

```
Usage of helm-secure-import:

Securely imports all images in a helm chart into a container registry.

Flags:
  -chart string
        Chart name (required)
  -registry string
        Destination registry URL (can also be set via HELM_REGISTRY env var)
  -repo string
        Repository URL (can be HTTP or OCI)
  -report-file string
        Report file (for json format)
  -report-format string
        Report format (table or json) (default "table")
  -sign-key string
        Signing key (optional)
  -values string
        Values file (optional)
  -version string
        Chart version (required)

Environment variables:
  HELM_REGISTRY    Registry URL (alternative to --registry flag)
  HELM_SIGN_KEY    Signing key path (alternative to --sign-key flag)
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

## Troubleshooting

### Common Issues

#### Authentication Errors
If you encounter authentication errors when connecting to your registry:
- Ensure you're logged in to your registry: `docker login <registry-url>`
- Verify your credentials have push permissions
- For private registries, ensure the registry URL format is correct

#### Missing Dependencies
If you get errors about missing tools:
- **Trivy**: Install from [https://aquasecurity.github.io/trivy/](https://aquasecurity.github.io/trivy/)
- **Copa**: Install from [https://github.com/project-copacetic/copacetic](https://github.com/project-copacetic/copacetic)
- **Cosign**: Install from [https://docs.sigstore.dev/cosign/installation/](https://docs.sigstore.dev/cosign/installation/)

#### Chart Not Found
If you see "chart not found" errors:
- Verify the chart name and version exist in the repository
- Check if the repository URL is accessible
- For OCI repositories, ensure the URL starts with `oci://`

#### Permission Denied
If you encounter permission errors:
- Check that your Docker daemon is running and accessible
- Verify you have write permissions to the destination registry
- Ensure your signing key file (if used) is readable

#### Network Issues
If you experience connectivity problems:
- Check your internet connection
- Verify firewall settings allow connections to the registry
- For corporate networks, check if proxy settings are needed

## Acknowledgements

The idea for this plugin is inspired in [Helmper](https://github.com/ChristofferNissen/helmper).

## License

This project is licensed under the MIT license.


