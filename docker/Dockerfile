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
#
# Usage:
#   docker run --rm -v $(pwd):/work -w /work kirbownz/combine-changelogs:latest \
#     combine-changelogs -project "mygroup/myrepo"

# Define build arguments
ARG ALPINE_VERSION=3.23
ARG VERSION
ARG BUILD_DATE
ARG VCS_REF
ARG TARGETARCH

################## Create the runtime image
FROM alpine:${ALPINE_VERSION} AS runner

ARG VERSION
ARG BUILD_DATE
ARG VCS_REF
ARG TARGETARCH

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

# Copy the pre-built binary from CI/CD artifacts.
# Owned by root, executable by all — users can run but not modify it.
COPY --chmod=755 combine-changelogs-linux-${TARGETARCH} /usr/local/bin/combine-changelogs
