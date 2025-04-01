# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Building environment
FROM golang:1.24.2-alpine@sha256:7772cb5322baa875edd74705556d08f0eeca7b9c4b5367754ce3f2f00041ccee AS build

WORKDIR /workspace

RUN apk add --upgrade --no-cache make=~4

# Copy everything and download deps
COPY . .

# Build binary
RUN go mod download && make build

FROM alpine:3.21@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

RUN apk add --upgrade --no-cache curl=~8

RUN addgroup -S sre && adduser -S sre -G sre
USER sre

COPY --from=build /workspace/build/metrics-exporter /metrics-exporter

ENTRYPOINT ["/metrics-exporter"]
