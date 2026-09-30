package agent

import (
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/nri-consul/src/metrics"
)

var gaugeMetrics = []*metrics.MetricDefinition{
	{
		APIKey:     "consul.runtime.num_goroutines",
		MetricName: "runtime.goroutines",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.alloc_bytes",
		MetricName: "runtime.allocationsInBytes",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.heap_objects",
		MetricName: "runtime.heapObjects",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.sys_bytes",
		MetricName: "runtime.virtualAddressSpaceInBytes",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.malloc_count",
		MetricName: "runtime.allocations",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.free_count",
		MetricName: "runtime.frees",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.total_gc_pause_ns",
		MetricName: "runtime.gcPauseInMilliseconds",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.runtime.total_gc_runs",
		MetricName: "runtime.gcCycles",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.agent.tls.cert.expiry",
		MetricName: "agent.tlsCertExpiryInSeconds",
		SourceType: metric.GAUGE,
	},
	{
		APIKey:     "consul.leaf-certs.entries_count",
		MetricName: "certs.leafCacheEntries",
		SourceType: metric.GAUGE,
	},
}

var counterMetrics = []*metrics.MetricDefinition{
	{
		APIKey:     "consul.client.rpc",
		MetricName: "client.rpcLoad",
		SourceType: metric.RATE,
	},
	{
		APIKey:     "consul.client.rpc.exceeded",
		MetricName: "client.rpcRateLimited",
		SourceType: metric.RATE,
	},
	{
		APIKey:     "consul.client.rpc.failed",
		MetricName: "client.rpcFailed",
		SourceType: metric.RATE,
	},
	{
		APIKey:     "consul.acl.cache_hit",
		MetricName: "agent.aclCacheHit",
		SourceType: metric.RATE,
	},
	{
		APIKey:     "consul.acl.cache_miss",
		MetricName: "agent.aclCacheMiss",
		SourceType: metric.RATE,
	},
	{
		APIKey:     "consul.dns.stale_queries",
		MetricName: "agent.staleQueries",
		SourceType: metric.RATE,
	},
}

var timerMetrics = []*metrics.TimerDefinition{
	{
		MetricDefinition: metrics.MetricDefinition{
			APIKey:     "consul.txn.apply",
			MetricName: "agent.txnAvgInMilliseconds",
			SourceType: metric.GAUGE,
		},
		Operation: metrics.Average,
	},
	{
		MetricDefinition: metrics.MetricDefinition{
			APIKey:     "consul.txn.apply",
			MetricName: "agent.txns",
			SourceType: metric.RATE,
		},
		Operation: metrics.Count,
	},
	{
		MetricDefinition: metrics.MetricDefinition{
			APIKey:     "consul.txn.apply",
			MetricName: "agent.txnMaxInMilliseconds",
			SourceType: metric.GAUGE,
		},
		Operation: metrics.Max,
	},
	{
		MetricDefinition: metrics.MetricDefinition{
			APIKey:     "consul.kvs.apply",
			MetricName: "agent.kvStoresAvgInMilliseconds",
			SourceType: metric.GAUGE,
		},
		Operation: metrics.Average,
	},
	{
		MetricDefinition: metrics.MetricDefinition{
			APIKey:     "consul.kvs.apply",
			MetricName: "agent.kvStoress",
			SourceType: metric.RATE,
		},
		Operation: metrics.Count,
	},
	{
		MetricDefinition: metrics.MetricDefinition{
			APIKey:     "consul.kvs.apply",
			MetricName: "agent.kvStoresMaxInMilliseconds",
			SourceType: metric.GAUGE,
		},
		Operation: metrics.Max,
	},
}
