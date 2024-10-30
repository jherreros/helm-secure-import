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
