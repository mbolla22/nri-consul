package agent

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-consul/src/args"
	"github.com/newrelic/nri-consul/src/testutils"
)

func TestCollectMetrics_CoreMetrics(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	arg := args.ArgumentList{
		Hostname:  hostname,
		Port:      port,
		EnableSSL: false,
		Timeout:   "0s",
	}

	apiConfig, err := arg.CreateAPIConfig(arg.Hostname)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	client, err := api.NewClient(apiConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	entity, err := i.Entity("test", "agent")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	agent := &Agent{
		Client:     client,
		entity:     entity,
		datacenter: "MyDC",
		ipAddr:     "192.168.0.0",
	}

	agents := []*Agent{agent}

	mux.HandleFunc("/v1/agent/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"Timestamp": "2018-10-26 14:17:50 +0000 UTC",
			"Gauges": [
				{
					"Name": "consul.runtime.free_count",
					"Value": 115177384,
					"Labels": {}
				},
				{
					"Name": "consul.runtime.heap_objects",
					"Value": 33463,
					"Labels": {}
				},
				{
					"Name": "consul.runtime.malloc_count",
					"Value": 115210850,
					"Labels": {}
				},
				{
					"Name": "consul.runtime.num_goroutines",
					"Value": 49,
					"Labels": {}
				},
				{
					"Name": "consul.runtime.sys_bytes",
					"Value": 14395640,
					"Labels": {}
				},
				{
					"Name": "consul.runtime.total_gc_pause_ns",
					"Value": 679636350,
					"Labels": {}
				},
				{
					"Name": "consul.runtime.total_gc_runs",
					"Value": 24701,
					"Labels": {}
				}
			],
			"Points": [],
			"Counters": [
				{
					"Name": "consul.acl.cache_hit",
					"Count": 2,
					"Rate": 0.2,
					"Sum": 2,
					"Min": 1,
					"Max": 1,
					"Mean": 1,
					"Stddev": 0,
					"Labels": {}
				}
			],
			"Samples": [
				{
					"Name": "consul.txn.apply",
					"Count": 1,
					"Rate": 0.16021829843521118,
					"Sum": 5,
					"Min": 1,
					"Max": 5,
					"Mean": 3,
					"Stddev": 0,
					"Labels": {}
				}
			]
		}`)
	})

	expected := map[string]interface{}{
		"event_type":                         "ConsulAgentSample",
		"displayName":                        agent.entity.Metadata.Name,
		"entityName":                         agent.entity.Metadata.Namespace + ":" + agent.entity.Metadata.Name,
		"datacenter":                         agent.datacenter,
		"ip":                                 agent.ipAddr,
		"runtime.goroutines":                 float64(49),
		"runtime.heapObjects":                float64(33463),
		"runtime.virtualAddressSpaceInBytes": float64(14395640),
		"runtime.allocations":                float64(115210850),
		"runtime.frees":                      float64(115177384),
		"runtime.gcPauseInMilliseconds":      float64(679636350) / 1000000,
		"runtime.gcCycles":                   float64(24701),
		"agent.aclCacheHit":                  float64(0),
		"agent.txnAvgInMilliseconds":         float64(3),
		"agent.txns":                         float64(0),
		"agent.txnMaxInMilliseconds":         float64(5),
	}

	CollectMetrics(agents, true)

	result := agent.entity.Metrics[0].Metrics
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v got %+v", expected, result)
	}
}

func TestCollectMetrics_PeerMetrics(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	arg := args.ArgumentList{
		Hostname:  hostname,
		Port:      port,
		EnableSSL: false,
		Timeout:   "0s",
	}

	apiConfig, err := arg.CreateAPIConfig(arg.Hostname)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	client, err := api.NewClient(apiConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	entity, err := i.Entity("test", "agent")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	agent := &Agent{
		Client:     client,
		entity:     entity,
		datacenter: "MyDC",
		ipAddr:     "192.168.0.0",
	}

	agents := []*Agent{agent}

	mux.HandleFunc("/v1/status/peers", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			"10.0.0.0:8300",
			"10.0.0.2:8300",
			"10.0.0.3:8300"
		]`)
	})

	expected := map[string]interface{}{
		"event_type":  "ConsulAgentSample",
		"displayName": agent.entity.Metadata.Name,
		"entityName":  agent.entity.Metadata.Namespace + ":" + agent.entity.Metadata.Name,
		"datacenter":  agent.datacenter,
		"ip":          agent.ipAddr,
		"agent.peers": float64(3),
	}

	CollectMetrics(agents, false)

	result := agent.entity.Metrics[0].Metrics
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v got %+v", expected, result)
	}
}

func TestCollectMetrics_LatencyMetrics(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	arg := args.ArgumentList{
		Hostname:  hostname,
		Port:      port,
		EnableSSL: false,
		Timeout:   "0s",
	}

	apiConfig, err := arg.CreateAPIConfig(arg.Hostname)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	client, err := api.NewClient(apiConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	memberNameIDAttr := integration.NewIDAttribute("co-agent", "consul-0")
	entity, err := i.Entity("consul-0", "agent", memberNameIDAttr)
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	agent := &Agent{
		Client:     client,
		entity:     entity,
		datacenter: "MyDC",
		ipAddr:     "192.168.0.0",
		name:       "consul-0",
	}

	agents := []*Agent{agent}

	mux.HandleFunc("/v1/coordinate/nodes", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"Node": "consul-0",
				"Segment": "",
				"Coord": {
					"Vec": [
						1,
						2,
						3
					],
					"Error": 0.08253992124813844,
					"Adjustment": -0.00003836604013478102,
					"Height": 0.0003466944852283233
				}
			},
			{
				"Node": "consul-1",
				"Segment": "",
				"Coord": {
					"Vec": [
						1,
						2,
						3
					],
					"Error": 0.06923646098980592,
					"Adjustment": -0.00002374826302067419,
					"Height": 0.00004579452297003133
				}
			},
			{
				"Node": "consul-2",
				"Segment": "",
				"Coord": {
					"Vec": [
						1,
						2,
						3
					],
					"Error": 0.08419609268393471,
					"Adjustment": -0.0000457158939839238,
					"Height": 0.00019087018135238138
				}
			},
			{
				"Node": "vault-0",
				"Segment": "",
				"Coord": {
					"Vec": [
						1,
						2,
						3
					],
					"Error": 0.08989722523185267,
					"Adjustment": -0.00002838544900300396,
					"Height": 0.00001
				}
			}
		]`)
	})

	expected := map[string]interface{}{
		"displayName": "consul-0",
		"entityName":  "agent:consul-0",
		"event_type":  "ConsulAgentSample",
		"datacenter":  agent.datacenter,
		"ip":          agent.ipAddr,

		"net.agent.medianLatencyInMilliseconds": 0.3303747050428994,
		"net.agent.minLatencyInMilliseconds":    0.28994299609053836,
		"net.agent.maxLatencyInMilliseconds":    0.453482732462,
		"net.agent.p75LatencyInMilliseconds":    0.453482732462,
		"net.agent.p95LatencyInMilliseconds":    0.453482732462,
		"net.agent.p25LatencyInMilliseconds":    0.28994299609053836,
		"net.agent.p90LatencyInMilliseconds":    0.453482732462,
		"net.agent.p99LatencyInMilliseconds":    0.453482732462,
	}

	CollectMetrics(agents, false)

	result := agent.entity.Metrics[0].Metrics
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v got %+v", expected, result)
	}
}
