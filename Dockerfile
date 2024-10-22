# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY ./cmd/secure-import .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o secure-import .

# Final stage
FROM alpine:3.20

# Install basic tools
RUN apk add --no-cache \
    bash \
    curl \
    docker-cli \
    git \
    yq \
    openssl

# Install helm
RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install Trivy
RUN curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Install Copa
RUN curl -LO https://github.com/project-copa/copa/releases/latest/download/copa-linux-amd64 && \
    chmod +x copa-linux-amd64 && \
    mv copa-linux-amd64 /usr/local/bin/copa

# Install Cosign
RUN curl -LO https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64 && \
    chmod +x cosign-linux-amd64 && \
    mv cosign-linux-amd64 /usr/local/bin/cosign

# Copy the compiled application
COPY --from=builder /app/secure-import /usr/local/bin/

ENTRYPOINT ["secure-import"]