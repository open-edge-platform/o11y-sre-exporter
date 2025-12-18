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

FROM alpine:3.23@sha256:be171b562d67532ea8b3c9d1fc0904288818bb36fc8359f954a7b7f1f9130fb2

RUN apk add --upgrade --no-cache curl=~8

RUN addgroup -S sre && adduser -S sre -G sre
USER sre

COPY --from=build /workspace/build/metrics-exporter /metrics-exporter

ENTRYPOINT ["/metrics-exporter"]
