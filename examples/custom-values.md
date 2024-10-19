# Custom Values Example

This example shows how to import a chart while using custom values to ensure all required images are captured.

## Prerequisites

Same as basic example, plus:
- Custom values file

## Custom Values File

Create a file named `values.yaml`:

```yaml
image:
  registry: docker.io
  repository: bitnami/nginx
  tag: 1.24.0-debian-11-r0

metrics:
  enabled: true
  image:
    registry: docker.io
    repository: bitnami/nginx-exporter
    tag: 0.11.0-debian-11-r0

sidecars:
  - name: proxy
    image: docker.io/bitnami/nginx-proxy:1.0.0
```

## Import Command

```bash
helm secure-import \
  --chart nginx \
  --version 15.0.2 \
  --repo-url https://charts.bitnami.com/bitnami \
  --registry my.registry.io \
  --sign-key ./cosign.key \
  --values ./values.yaml
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
- docker.io/bitnami/nginx-exporter:0.11.0-debian-11-r0
- docker.io/bitnami/nginx-proxy:1.0.0
Processing images...
Scanning for vulnerabilities...
Found vulnerabilities in nginx-proxy:1.0.0, patching...
Patch completed successfully
Pushing images to registry...
All images pushed successfully
Signing images...
All images signed successfully
```
