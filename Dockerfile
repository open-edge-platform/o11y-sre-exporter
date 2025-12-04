# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Building environment
FROM golang:1.25.5-alpine AS build

WORKDIR /workspace

RUN apk add --upgrade --no-cache make=~4 bash=~5

# Copy everything and download deps
COPY . .

# Build binary
RUN go mod download && make build

FROM alpine:3.23@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375

RUN apk add --upgrade --no-cache curl=~8

RUN addgroup -S sre && adduser -S sre -G sre
USER sre

COPY --from=build /workspace/build/metrics-exporter /metrics-exporter

ENTRYPOINT ["/metrics-exporter"]
