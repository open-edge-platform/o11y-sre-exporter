<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# SRE Exporter Changelog

## [v0.8.22](https://github.com/open-edge-platform/o11y-sre-exporter/tree/v0.8.22)

- Initial release
- Application `sre-exporter` added:
  - Deployable via Helm Chart
  - Supports [External Secrets](https://external-secrets.io/latest/) for external destination endpoint configuration
  - Exports data via [Prometheus Remote-Write Protocol](https://prometheus.io/docs/specs/remote_write_spec/)
  - Processing and export done via dependent [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
  - Multitenancy support with management endpoint exposed via `gRPC`
  - Configurable metric collection from `Grafana Mimir` and `Vault` instances
