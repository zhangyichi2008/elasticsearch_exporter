# Elasticsearch Exporter
[![CircleCI](https://circleci.com/gh/prometheus-community/elasticsearch_exporter.svg?style=svg)](https://circleci.com/gh/prometheus-community/elasticsearch_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/prometheus-community/elasticsearch_exporter)](https://goreportcard.com/report/github.com/prometheus-community/elasticsearch_exporter)

Prometheus exporter for various metrics about Elasticsearch, written in Go.

**1.5.0 / 2023-09-25**


**[ADD] 增加对Node节点总分片个数的监控**
- HELP elasticsearch_node_shards_total Total shards per node
- TYPE elasticsearch_node_shards_total gauge
- elasticsearch_node_shards_total{node="xt01"} 243


**[ADD] 增加对索引主分片大小(单位:GB)的监控**

- HELP elasticsearch_indices_primary_shares_size_gbytes Current total size of stored index data in gbytes with only primary shards on all nodes
- TYPE elasticsearch_indices_primary_shares_size_gbytes gauge
- elasticsearch_indices_primary_shares_size_gbytes{cluster="elk",index="tomca-"} 5.93


**[ADD] 增加对es分配内存大小(单位:GB)的监控**

- HELP elasticsearch_jvm_memory_max_gbytes JVM memory max
- TYPE elasticsearch_jvm_memory_max_gbytes gauge
- elasticsearch_jvm_memory_max_gbytes{area="heap",cluster="elk",host="xt01",name="xt01"} 31


**[DEL] 移除以go_开头的33个监控项,具体如下：**

- go_gc_duration_seconds{quantile="0"}
- go_gc_duration_seconds{quantile="0.25"}
- go_gc_duration_seconds{quantile="0.5"}
- go_gc_duration_seconds{quantile="0.75"}
- go_gc_duration_seconds{quantile="1"}
- go_gc_duration_seconds_sum
- go_gc_duration_seconds_count
- go_goroutines
- go_info{version="go1.19.1"}
- go_memstats_alloc_bytes
- go_memstats_alloc_bytes_total
- go_memstats_buck_hash_sys_bytes
- go_memstats_frees_total
- go_memstats_gc_sys_bytes
- go_memstats_heap_alloc_bytes
- go_memstats_heap_idle_bytes
- go_memstats_heap_inuse_bytes
- go_memstats_heap_objects
- go_memstats_heap_released_bytes
- go_memstats_heap_sys_bytes
- go_memstats_last_gc_time_seconds
- go_memstats_lookups_total
- go_memstats_mallocs_total
- go_memstats_mcache_inuse_bytes
- go_memstats_mcache_sys_bytes
- go_memstats_mspan_inuse_bytes
- go_memstats_mspan_sys_bytes
- go_memstats_next_gc_bytes
- go_memstats_other_sys_bytes
- go_memstats_stack_inuse_bytes
- go_memstats_stack_sys_bytes
- go_memstats_sys_bytes
- go_threads


### Configuration

**NOTE:** The exporter fetches information from an Elasticsearch cluster on every scrape, therefore having a too short scrape interval can impose load on ES master nodes, particularly if you run with `--es.all` and `--es.indices`. We suggest you measure how long fetching `/_nodes/stats` and `/_all/_stats` takes for your ES cluster to determine whether your scraping interval is too short. As a last resort, you can scrape this exporter using a dedicated job with its own scraping interval.

Below is the command line options summary:
```bash
elasticsearch_exporter --help
```

| Argument                | Introduced in Version | Description | Default     |
| --------                | --------------------- | ----------- | ----------- |
| es.uri                  | 1.0.2                 | Address (host and port) of the Elasticsearch node we should connect to. This could be a local node (`localhost:9200`, for instance), or the address of a remote Elasticsearch server. When basic auth is needed, specify as: `<proto>://<user>:<password>@<host>:<port>`. E.G., `http://admin:pass@localhost:9200`. Special characters in the user credentials need to be URL-encoded. | http://localhost:9200 |
| es.all                  | 1.0.2                 | If true, query stats for all nodes in the cluster, rather than just the node we connect to.                             | false |
| es.cluster_settings     | 1.1.0rc1              | If true, query stats for cluster settings. | false |
| es.indices              | 1.0.2                 | If true, query stats for all indices in the cluster. | false |
| es.indices_settings     | 1.0.4rc1              | If true, query settings stats for all indices in the cluster. | false |
| es.indices_mappings     | 1.2.0                 | If true, query stats for mappings of all indices of the cluster. | false |
| es.aliases              | 1.0.4rc1              | If true, include informational aliases metrics. | true |
| es.shards               | 1.0.3rc1              | If true, query stats for all indices in the cluster, including shard-level stats (implies `es.indices=true`). | false |
| es.snapshots            | 1.0.4rc1              | If true, query stats for the cluster snapshots. | false |
| es.slm                  |                       | If true, query stats for SLM. | false |
| es.data_stream          |                       | If true, query state for Data Steams. | false |
| es.timeout              | 1.0.2                 | Timeout for trying to get stats from Elasticsearch. (ex: 20s) | 5s |
| es.ca                   | 1.0.2                 | Path to PEM file that contains trusted Certificate Authorities for the Elasticsearch connection. | |
| es.client-private-key   | 1.0.2                 | Path to PEM file that contains the private key for client auth when connecting to Elasticsearch. | |
| es.client-cert          | 1.0.2                 | Path to PEM file that contains the corresponding cert for the private key to connect to Elasticsearch. | |
| es.clusterinfo.interval | 1.1.0rc1              |  Cluster info update interval for the cluster label | 5m |
| es.ssl-skip-verify      | 1.0.4rc1              | Skip SSL verification when connecting to Elasticsearch. | false |
| web.listen-address      | 1.0.2                 | Address to listen on for web interface and telemetry. | :9114 |
| web.telemetry-path      | 1.0.2                 | Path under which to expose metrics. | /metrics |
| version                 | 1.0.2                 | Show version info on stdout and exit. | |

Commandline parameters start with a single `-` for versions less than `1.1.0rc1`.
For versions greater than `1.1.0rc1`, commandline parameters are specified with `--`.

The API key used to connect can be set with the `ES_API_KEY` environment variable.

#### Elasticsearch 7.x security privileges

Username and password can be passed either directly in the URI or through the `ES_USERNAME` and `ES_PASSWORD` environment variables.
Specifying those two environment variables will override authentication passed in the URI (if any).

ES 7.x supports RBACs. The following security privileges are required for the elasticsearch_exporter.

Setting | Privilege Required | Description
:---- | :---- | :----
exporter defaults | `cluster` `monitor` | All cluster read-only operations, like cluster health and state, hot threads, node info, node and cluster stats, and pending cluster tasks. |
es.cluster_settings | `cluster` `monitor` |
es.indices | `indices` `monitor` (per index or `*`) | All actions that are required for monitoring (recovery, segments info, index stats and status)
es.indices_settings | `indices` `monitor` (per index or `*`) |
es.shards | not sure if `indices` or `cluster` `monitor` or both |
es.snapshots | `cluster:admin/snapshot/status` and `cluster:admin/repository/get` | [ES Forum Post](https://discuss.elastic.co/t/permissions-for-backup-user-with-x-pack/88057)
es.slm | `read_slm`
es.data_stream | `monitor` or `manage` (per index or `*`) |

Further Information
- [Build in Users](https://www.elastic.co/guide/en/elastic-stack-overview/7.3/built-in-users.html)
- [Defining Roles](https://www.elastic.co/guide/en/elastic-stack-overview/7.3/defining-roles.html)
- [Privileges](https://www.elastic.co/guide/en/elastic-stack-overview/7.3/security-privileges.html)


### Alerts & Recording Rules

We provide examples for [Prometheus](http://prometheus.io) [alerts and recording rules](examples/prometheus/elasticsearch.rules) as well as an [Grafana](http://www.grafana.org) [Dashboard](examples/grafana/dashboard.json) and a [Kubernetes](http://kubernetes.io) [Deployment](examples/kubernetes/deployment.yml).

The example dashboard needs the [node_exporter](https://github.com/prometheus/node_exporter) installed. In order to select the nodes that belong to the Elasticsearch cluster, we rely on a label `cluster`.
Depending on your setup, it can derived from the platform metadata:

For example on [GCE](https://cloud.google.com)

```
- source_labels: [__meta_gce_metadata_Cluster]
  separator: ;
  regex: (.*)
  target_label: cluster
  replacement: ${1}
  action: replace
```

Please refer to the [Prometheus SD documentation](https://prometheus.io/docs/operating/configuration/) to see which metadata labels can be used to create the `cluster` label.
