<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Development and Testing

This document shows how to deploy and test Edge Orchestrator with a custom version of `sre-exporter`.

[Edge Manageability Framework]: https://github.com/open-edge-platform/edge-manageability-framework

## Prerequisites

1. Clone the `sre-exporter` and [Edge Manageability Framework] repositories in the same folder:

   ```bash
   git clone https://github.com/open-edge-platform/edge-manageability-framework.git
   git clone https://github.com/open-edge-platform/o11y-sre-exporter.git
   ```

   It is important to preserve the original names of the repository folders.

1. Deploy the [Edge Manageability Framework] on Kind.

   Alternatively, already deployed cluster of Edge Orchestrator can be used for this purpose.

1. Create and publish your development branch of `sre-exporter` as a remote branch.

## Development of the `sre-exporter` application

### Deploying Edge Orchestrator with the custom version of `sre-exporter` chart

1. Create a new development branch of the Edge Manageability Framework repository.

1. Modify the sre-exporter application template.

   After checking out your development branch of `sre-exporter`, run the following command in `sre-exporter` repository root folder:

   ```bash
   # to be run in `sre-exporter` repo
   mage argo:updateSreTemplate
   ```

   This target automatically applies the following changes to the template `argocd/applications/templates/sre-exporter.yaml`
   under `.spec.sources[0]`:

    - Removed lines:

      ```yaml
      - repoURL: {{ required "A valid chartRepoURL entry required!" .Values.argo.chartRepoURL }}
        chart: o11y/charts/{{$appName}}
        targetRevision: <current sre-exporter revision>
      ```

    - Added lines:

       ```yaml
       - repoURL: https://github.com/open-edge-platform/o11y-sre-exporter
         path: deployments/sre-exporter
         targetRevision: <your dev branch in sre-exporter repo>
       ```

   As a result, ArgoCD will sync the `sre-exporter` application chart directly with the development branch.

1. Add `sre-exporter` repository to ArgoCD

   When ArgoCD itself is installed, log in to the Argo web UI and add a new repository.

   To do that, go to `settings` -> `repositories` -> `connect repo`.
   Choose `https` as the connection method, paste the `sre-exporter` repo URL, your GitHub token, and connect.

   Alternatively, the same can be achieved using the ArgoCD CLI.

    ```bash
    # to be run in `edge-manageability-framework` repo
    argocd login <External IP of ArgoCD server>:443 --username admin --password <ArgoCD admin password> --insecure
    argocd repo add https://github.com/open-edge-platform/o11y-sre-exporter --username $GITHUB_USER --password $GITHUB_API_TOKEN
    ```

1. Deploy the local version of Edge Orchestrator

   - Commit changes to the development branch of Edge Manageability Framework
   - Push the branch to the remote repository
   - Deploy the local changes using the `mage deploy:orchLocal` target (e.g. `mage deploy:orchLocal dev`). Refer to the documentation in the
   [Edge Manageability Framework] repository for more details.

   After several minutes, `sre-exporter` should get installed in the custom version tracking your remote branch of the `sre-exporter` repository.
   Each time you make changes to that branch and push them, Argo will notice the new commit and sync the changes made.

### Modifying the `sre-exporter` chart values

To update values with which `sre-exporter` will be deployed, the easiest way is to change the values in `argocd/applications/{configs|custom}/sre-exporter.{yaml|tpl}` paths of the [Edge Manageability Framework] repository.
You can overwrite every field there by simply specifying the value. Remember that custom values overwrite base values.

After changing the values, you need to push the changes to the remote branch and make ArgoCD sync to the new commit by running the `mage deploy:orchLocal` target.

For example, to install `sre-exporter` with `devMode` enabled, you'd modify the `argocd/applications/configs/sre-exporter.yaml` by simply adding `devMode: true` at the end.

**Note**: The `devMode` setting exposes testing ports in `metrics-exporter` and `otel-collector` containers that allow checking the presence of metrics and debugging the internal state of the `sre-exporter` application.

### Updating the set of exported metrics

The metrics exported are defined in the files:

- [sre-exporter-orch.json](../deployments/sre-exporter/files/configs/sre-exporter-orch.json) defines the metrics exported from the Edge Orchestrator Cluster
- [sre-exporter-edge-node.json](../deployments/sre-exporter/files/configs/sre-exporter-edge-node.json) defines the metrics exported from the Edge Nodes

Add, remove, or change `collectors[*].metrics[*]` elements in the JSON files to add, remove, or modify the metrics exported.
Update `collectors[*].metrics[*].query` to change the query used to collect the metric.

After modifying the exported metrics, remember to update the documentation with the command:

```bash
mage doc:generate
```

and commit the generated changes.

### Deploying the custom version of the `sre-exporter` application

The installation previously mentioned will only update the Helm chart components, but not the code changes in the application.

If that chart requires a new container image, where changes in the code have been made, you need to build the image and load it into Kind.

Follow the steps below to do that:

1. Update the [`VERSION`](../VERSION) file to something unique (e.g. add `-dev` suffix).

1. Run `make kind-all` to build and load the image.

1. Commit and push the changes to remote branch. Argo will now try to sync and will use the newly created image.

## Testing of the `sre-exporter` application

All the commands in this section should be run from the root folder of the `edge-manageability-framework` repository.

### Prerequisite

Before running the tests, deploy Victoria Metrics instance, which will serve as a destination for exported metrics:

   ```bash
   # to be run in `edge-manageability-framework` repo
   mage deploy:victoriaMetrics apply
   ```

### Testing Edge Orchestrator metrics

Run the following command to test the presence of metrics exported from Edge Orchestrator cluster:

   **Note**: Wait a few minutes after deploying Victoria Metrics before running the test, so that all the metrics are exported to the destination.

   ```bash
   # to be run in `edge-manageability-framework` repo
   mage test:e2eSreObservabilityNoEnic
   ```

### Testing Edge Node metrics

Run the following commands to test the presence of metrics exported from Edge Node deployed as ENiC (Edge Node in container):

**Note**: Wait a few minutes after deploying Edge Node before running the test, so that all the metrics are exported to the destination.

- Deploy ENiC

  ```bash
  # to be run in `edge-manageability-framework` repo
  mage devUtils:deployEnic 1 dev
  ```

- Run the tests

  ```bash
  # to be run in `edge-manageability-framework` repo
  mage test:e2eSreObservability
  ```

### Clean up after tests

After running the tests, delete the Victoria Metrics and ENiC instances.

   ```bash
   # to be run in `edge-manageability-framework` repo
   mage deploy:victoriaMetrics delete
   mage devUtils:deployEnic 0 dev
   ```

### Troubleshooting the `sre-exporter` application

Please refer to the troubleshooting guides in the Edge Orchestrator documentation for details on how to diagnose and resolve issues related to the `sre-exporter` application.
