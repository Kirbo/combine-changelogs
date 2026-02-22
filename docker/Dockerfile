# syntax=docker/dockerfile:latest

### Build and publish the Docker image
# docker buildx build \
#   --platform linux/arm64,linux/amd64 \
#   --build-arg VERSION=$(git describe --tags --always) \
#   --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
#   --build-arg VCS_REF=$(git rev-parse --short HEAD) \
#   -t kirbownz/combine-changelogs:latest \
#   -t kirbownz/combine-changelogs:$(git describe --tags --always) \
#   --push \
#   -f docker/Dockerfile .

# Check the base images from: https://hub.docker.com/hardened-images/catalog/dhi/alpine-base/images

# Define build arguments
ARG ALPINE_VERSION=3.23
ARG VERSION
ARG BUILD_DATE
ARG VCS_REF

################## Create the build image
FROM dhi.io/alpine-base:${ALPINE_VERSION}-dev AS builder

ARG VERSION

# Install curl for downloading the binary
RUN apk add --no-cache curl

# Download and install combine-changelogs
# VERSION is empty → installs the latest release; set VERSION=vX.Y.Z to pin a specific one
RUN VERSION="${VERSION}" curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh

################## Create the runtime image
FROM dhi.io/alpine:${ALPINE_VERSION} AS runner

ARG VERSION
ARG BUILD_DATE
ARG VCS_REF

# OCI Image Format Specification labels (https://github.com/opencontainers/image-spec/blob/main/annotations.md)
LABEL org.opencontainers.image.title="combine-changelogs"
LABEL org.opencontainers.image.description="CLI tool that combines releases from a GitLab project with local or remote changelog files into a single CHANGELOG.md"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.authors="Kimmo Saari <kirbo@kirbo-designs.com>"
LABEL org.opencontainers.image.url="https://gitlab.com/kirbo/combine-changelogs"
LABEL org.opencontainers.image.documentation="https://gitlab.com/kirbo/combine-changelogs/-/blob/main/README.md"
LABEL org.opencontainers.image.source="https://gitlab.com/kirbo/combine-changelogs"
LABEL org.opencontainers.image.revision="${VCS_REF}"
LABEL org.opencontainers.image.vendor="Kimmo Saari"
LABEL org.opencontainers.image.licenses="MIT"

LABEL maintainer="Kimmo Saari <kirbo@kirbo-designs.com>"
LABEL description="Combines GitLab releases and changelog files into a single CHANGELOG.md"

# Copy the statically-linked binary from the builder stage
COPY --from=builder /usr/local/bin/combine-changelogs /usr/local/bin/combine-changelogs

ENTRYPOINT ["combine-changelogs"]
