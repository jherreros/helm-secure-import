# Basic Example

This example demonstrates how to import the NGINX chart from the Bitnami repository to your OCI-compliant Registry.

## Simple scenario

### Prerequisites

- OCI-compliant Registry access

### Usage

Import the NGINX chart:

```bash
helm secure-import nginx \
  --version 18.2.4 \
  --repo https://charts.bitnami.com/bitnami \
  --registry my.registry.io
```

### With registry as environment variable


```bash
export HELM_REGISTRY=my.registry.io
helm secure-import nginx \
  --version 18.2.4 \
  --repo https://charts.bitnami.com/bitnami \
```

## Signing images

In addition to importing a chart and the images in it, you can sign them.

### Prerequisites

Same as simple scenario, plus:
- Cosign key pair (if you don't have one, create it with `cosign generate-key-pair`)

### Usage

```bash
helm secure-import nginx \
  --version 18.2.4 \
  --repo https://charts.bitnami.com/bitnami \
  --registry my.registry.io \
  --sign-key ./cosign.key
```

## Using a Helm values file 

To ensure all required images in a chart are extracted, you might need to provide your Helm values file.

## Prerequisites

Same as basic example, plus:
- [Custom values file](./values.yaml)

## Usage

```bash
helm secure-import nginx \
  --version 18.2.4 \
  --repo https://charts.bitnami.com/bitnami \
  --registry my.registry.io \
  --sign-key ./cosign.key \
  --values ./values.yaml
```
