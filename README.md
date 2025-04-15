<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Edge Orchestrator SRE Exporter

[Documentation]: https://github.com/open-edge-platform/orch-docs
[Prometheus Remote-Write Protocol]: https://prometheus.io/docs/specs/remote_write_spec/
[Exported metrics specification]: docs/exported-metrics-spec.md
[Contributor's Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[Edge Orchestrator Community]: https://github.com/open-edge-platform
[Troubleshooting]: https://github.com/open-edge-platform/orch-docs
[Contact us]: https://github.com/open-edge-platform

[Apache 2.0 License]: LICENSES/Apache-2.0.txt

## Overview

The SRE Exporter is a service running in Edge Orchestrator that perpetually collects a subset of metrics from both Edge Orchestrator services and Edge Nodes, and exports them to an arbitrary external destination server using the [Prometheus Remote-Write Protocol].

The SRE Exporter is intended to enhance the reliability of Edge Orchestrator by allowing it to store and monitor its essential metrics within an independent Site Reliability Engineering (SRE) system. Such an SRE system may implement its own data retention policy and provide SRE functions like performance monitoring, incident management, on-call management, etc.

The SRE Exporter allows for the configuration of the following settings:

- Destination server URL and basic authentication credentials
- TLS (optionally the server's private CA root certificate can be provided)
- Export time interval
- Edge Orchestrator instance-specific label

The SRE Exporter allows for monitoring the following metrics:

- Edge Orchestrator API success rate and latency
- Edge Orchestrator cluster nodes saturation metrics (CPU and memory)
- Edge Orchestrator Vault status
- Edge Nodes metrics (CPU, memory, disk, temperature) across all organizations and projects

See the [Exported metrics specification] for more details.

Read more about the SRE Exporter in the [Documentation].

## Get Started

To set up the development environment and work on this project, follow the steps below.
All necessary tools will be installed using the `install-tools` target.
Note that `docker` and `asdf` must be installed beforehand.

### Install Tools

To install all the necessary tools needed for development the project, run:

```sh
make install-tools
```

### Build

To build the project, use the following command:

```sh
make build
```

### Lint

To lint the code and ensure it adheres to the coding standards, run:

```sh
make lint
```

### Test

To run the tests and verify the functionality of the project, use:

```sh
make test
```

### Docker Build

To build the Docker images for the project, run:

```sh
make docker-build
```

### Helm Build

To package the Helm chart for the project, use:

```sh
make helm-build
```

### Docker Push

To push the Docker images to the registry, run:

```sh
make docker-push
```

### Helm Push

To push the Helm chart to the repository, use:

```sh
make helm-push
```

### Kind All

To load the Docker images into a local Kind cluster, run:

```sh
make kind-all
```

### Proto

To generate code from protobuf definitions, use:

```sh
make proto
```

## Develop

It is recommended to develop the `sre-exporter` application by deploying and testing it as a part of the Edge Orchestrator cluster.
Refer to [Development and Testing](docs/develop.md) document for more detailed instructions.

The code of this project is maintained and released in CI using the `VERSION` file.
In addition, the chart is versioned with the same tag as the `VERSION` file.

This is mandatory to keep all chart versions and app versions coherent.

To bump the version, increment the version in the `VERSION` file and run the following command
(to set `version` and `appVersion` in the `Chart.yaml` automatically):

```sh
make helm-build
```

## Contribute

To learn how to contribute to the project, see the [Contributor's Guide].

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community].

For support, start with [Troubleshooting] or [Contact us].

## License

The Edge Orchestrator Site Reliability Engineering (SRE) Exporter is licensed under the [Apache 2.0 License].

Last Updated Date: {March 28, 2025}
