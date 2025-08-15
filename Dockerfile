# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Building environment
FROM golang:1.25.0-alpine AS build

WORKDIR /workspace

RUN apk add --upgrade --no-cache make=~4 bash=~5

# Copy everything and download deps
COPY . .

# Build binary
RUN go mod download && make build

FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

RUN apk add --upgrade --no-cache curl=~8

RUN addgroup -S sre && adduser -S sre -G sre
USER sre

COPY --from=build /workspace/build/metrics-exporter /metrics-exporter

ENTRYPOINT ["/metrics-exporter"]
