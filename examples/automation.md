# Automation Example

This example demonstrates how to automate chart imports using a shell script.

## Prerequisites

Same as basic example, plus:
- Charts list file

## Charts Configuration

Create a file named `charts.yaml`:

```yaml
charts:
  - name: nginx
    version: 15.0.2
    repository: https://charts.bitnami.com/bitnami
    values: ./nginx-values.yaml
  - name: postgresql
    version: 12.5.3
    repository: https://charts.bitnami.com/bitnami
    values: ./postgresql-values.yaml
  - name: redis
    version: 17.11.3
    repository: https://charts.bitnami.com/bitnami
    values: ./redis-values.yaml
```

## Automation Script

Create a file named `import-charts.sh`:

```bash
#!/usr/bin/env bash

set -e

# Configuration
REGISTRY="my.registry.io"
SIGN_KEY="./cosign.key"
CHARTS_FILE="./charts.yaml"

# Read and process charts
echo "Processing charts from $CHARTS_FILE"
charts=($(yq e '.charts[] | [.name, .version, .repository, .values] | join(" ")' "$CHARTS_FILE"))

for chart in "${charts[@]}"; do
    read -r name version repo values <<< "$chart"
    
    echo "Importing chart: $name:$version"
    
    args=(
        --chart "$name"
        --version "$version"
        --repo-url "$repo"
        --registry "$REGISTRY"
        --sign-key "$SIGN_KEY"
    )
    
    if [ -f "$values" ]; then
        args+=(--values "$values")
    fi
    
    helm secure-import "${args[@]}"
done

echo "All charts imported successfully!"
```

Make the script executable:
```bash
chmod +x import-charts.sh
```

## Usage

Run the automation script:
```bash
./import-charts.sh
```

This will process each chart in the configuration file, importing it and its images to your registry.