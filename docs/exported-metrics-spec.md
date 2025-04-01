# Documentation of exported metrics in sre-exporter (v0.8.7)

## 1. Exported Orchestrator Metrics

<!-- Begin of auto-generated Markdown table -->
Name | Type | Description | Constant labels | Variable labels | Query
:---: | :---: | :---: | :---: | :---: | :---:
orch_IstioCollector_istio_requests | Counter | The total count of HTTP/GRPC requests processed by Istio Proxies | service, customer | connection_security_policy, destination_app, destination_canonical_revision, destination_canonical_service, destination_cluster, destination_principal, destination_service, destination_service_name, destination_service_namespace, destination_version, destination_workload, destination_workload_namespace, instance, job, pod, pod_name, reporter, request_protocol, response_code, response_flags, source_app, source_canonical_revision, source_canonical_service, source_cluster, source_principal, source_version, source_workload, source_workload_namespace, grpc_response_status | istio_requests_total{destination_workload=~'app-deployment-api\|app-resource-manager\|catalog-service\|cluster-template-manager\|ecm\|harbor-chartmuseum\|harbor-core\|harbor-jobservice\|harbor-portal\|harbor-registry\|harbor-trivy\|api\|inventory\|platform-keycloak\|rancher\|vault'}
orch_NodeCollector_cpu_total_cores | Gauge | Total CPU cores per node | service, customer | k8s_node_name | sum by(k8s_node_name) (k8s_node_allocatable_cpu)
orch_NodeCollector_cpu_used_cores | Gauge | Used CPU cores per node | service, customer | k8s_node_name | sum by(k8s_node_name) (k8s_node_cpu_utilization)
orch_NodeCollector_memory_total_bytes | Gauge | Total memory per node in Bytes | service, customer | k8s_node_name | k8s_node_allocatable_memory
orch_NodeCollector_memory_available_bytes | Gauge | Current available memory per node in Bytes | service, customer | k8s_node_name | k8s_node_memory_available
orch_api_requests_all | Counter | The total count of HTTP request processed | service, customer | status, target_service, method, protocol, gw_instance, gw_namespace, gw_pod | traefik_service_requests_total
orch_api_request_latency_seconds_all | Counter | Histogram of HTTP request latencies | service, customer | status, target_service, method, protocol, gw_instance, gw_namespace, gw_pod, le | traefik_service_request_duration_seconds_bucket
<!-- End of auto-generated Markdown table -->

## 2. Exported Edge Node Metrics

<!-- Begin of auto-generated Markdown table -->
Name | Type | Description | Constant labels | Variable labels | Query
:---: | :---: | :---: | :---: | :---: | :---:
orch_edgenode_env_temp | Gauge | Temperature of a sensor on Edge Node host [Celsius] | service, customer | host, hostGuid, projectId, sensor | temp_temp
orch_edgenode_mem_used_percent | Gauge | Percentage of used memory vs total memory on Edge Node host | service, customer | host, hostGuid, projectId | mem_used_percent
orch_edgenode_disk_used_percent | Gauge | Percentage of used vs total available space on a disk of Edge Node host | service, customer | host, hostGuid, projectId, device, path, mode | disk_used_percent
orch_edgenode_cpu_idle_percent | Gauge | Percentage of idle vs total CPU cycles on Edge Node host | service, customer | host, hostGuid, projectId | cpu_usage_idle{cpu='cpu-total'}
<!-- End of auto-generated Markdown table -->

## 3. Exported Vault Metrics

<!-- Begin of auto-generated Markdown table -->
Name | Type | Description | Constant labels | Variable labels | Query
:---: | :---: | :---: | :---: | :---: | :---:
orch_vault_monitor_vault_status | Gauge | The current status of vault instance ready:0 , sealed:1 , standby:2 | service, customer | k8s_pod_name | n/a (GET status)
<!-- End of auto-generated Markdown table -->
