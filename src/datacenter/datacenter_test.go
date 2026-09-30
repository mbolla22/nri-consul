package datacenter

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-consul/src/agent"
	"github.com/newrelic/nri-consul/src/args"
	"github.com/newrelic/nri-consul/src/testutils"
)

func TestNewDatacenter_NoLeader(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	if _, err := NewDatacenter(nil, i); err == nil {
		t.Error("Expected error")
	}

}
func TestNewDatacenter_Normal(t *testing.T) {
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

	agent := &agent.Agent{
		Client: client,
	}

	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"Config": {
				"Datacenter": "dev-uss",
				"NodeName": "vault-dev-uss-0",
				"NodeID": "70ccd111-96de-a058-7be7-81a00fa7ca17",
				"Revision": "39f93f011",
				"Server": false,
				"Version": "1.2.1"
			}
		}`)
	})

	out, err := NewDatacenter(agent, i)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if out.entity.Metadata.Name != "dev-uss" {
		t.Fatalf("Entity was not named correctly %s", out.entity.Metadata.Name)
	} else if out.entity.Metadata.Namespace != "co-datacenter" {
		t.Fatalf("Entity has wrong namespace %s", out.entity.Metadata.Namespace)
	}

	if out.leader != agent {
		t.Errorf("Agent was not set correctly expected %+v got %+v", agent, out.leader)
	}

}

func TestNewDatacenter_DCName_Config_Failure(t *testing.T) {
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

	agent := &agent.Agent{
		Client: client,
	}

	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"DebugConfig": {}
		}`)
	})

	if _, err := NewDatacenter(agent, i); err == nil {
		t.Error("Expected error")
	}
}

func TestNewDatacenter_DCName_Failure(t *testing.T) {
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

	agent := &agent.Agent{
		Client: client,
	}

	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"Config": {}
		}`)
	})

	if _, err := NewDatacenter(agent, i); err == nil {
		t.Error("Expected error")
	}
}

func TestNewDatacenter_Leader_Self_Failure(t *testing.T) {
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

	agent := &agent.Agent{
		Client: client,
	}

	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	if _, err := NewDatacenter(agent, i); err == nil {
		t.Error("Expected error")
	}
}

func Test_Datacenter_CollectMetrics_Full(t *testing.T) {
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

	dcEntity, err := i.Entity("test", "datacenter")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	agentEntity, err := i.Entity("leader", "agent")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	c := &Datacenter{
		entity: dcEntity,
		leader: agent.NewAgent(client, agentEntity, "", "", ""),
	}

	setMetricMuxes(mux)

	expected := map[string]interface{}{
		"event_type":                       "ConsulDatacenterSample",
		"displayName":                      c.entity.Metadata.Name,
		"entityName":                       c.entity.Metadata.Namespace + ":" + c.entity.Metadata.Name,
		"leader":                           "leader",
		"raft.txns":                        float64(0),
		"raft.commitTimeAvgInMilliseconds": float64(3),
		"raft.commitTimes":                 float64(0),
		"raft.commitTimeMaxInMilliseconds": float64(5),
		"catalog.registeredNodes":          float64(3),
		"catalog.criticalNodes":            float64(1),
		"catalog.upNodes":                  float64(1),
		"catalog.warningNodes":             float64(1),
		"catalog.passingNodes":             float64(1),
	}

	c.CollectMetrics(false)

	result := c.entity.Metrics[0].Metrics
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v got %+v", expected, result)
	}
}

func Test_Datacenter_CollectMetrics_All_Endpoint_Fails(t *testing.T) {
	_, hostname, port, serverClose := testutils.SetupServer()
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

	dcEntity, err := i.Entity("test", "datacenter")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	agentEntity, err := i.Entity("leader", "agent")
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}

	c := &Datacenter{
		entity: dcEntity,
		leader: agent.NewAgent(client, agentEntity, "", "", ""),
	}

	expected := map[string]interface{}{
		"event_type":  "ConsulDatacenterSample",
		"displayName": c.entity.Metadata.Name,
		"entityName":  c.entity.Metadata.Namespace + ":" + c.entity.Metadata.Name,
		"leader":      "leader",
	}

	c.CollectMetrics(false)

	result := c.entity.Metrics[0].Metrics
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v got %+v", expected, result)
	}
}

func setMetricMuxes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/agent/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"Timestamp": "2018-10-26 14:17:50 +0000 UTC",
			"Gauges": [],
			"Points": [],
			"Counters": [
				{
					"Name": "consul.raft.apply",
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
					"Name": "consul.raft.commitTime",
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

	mux.HandleFunc("/v1/catalog/nodes", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"ID": "1234",
				"Node": "consul-dev-0",
				"Address": "10.0.0.140",
				"Datacenter": "dev",
				"TaggedAddresses": {
					"lan": "10.0.0.140",
					"wan": "10.0.0.140"
				},
				"Meta": {
					"consul-network-segment": ""
				},
				"CreateIndex": 8,
				"ModifyIndex": 9
			},
			{
				"ID": "5678",
				"Node": "consul-dev-1",
				"Address": "10.0.0.142",
				"Datacenter": "dev",
				"TaggedAddresses": {
					"lan": "10.0.0.142",
					"wan": "10.0.0.142"
				},
				"Meta": {
					"consul-network-segment": ""
				},
				"CreateIndex": 65,
				"ModifyIndex": 67
			},
			{
				"ID": "91011",
				"Node": "vault-dev-1",
				"Address": "10.0.0.55",
				"Datacenter": "dev",
				"TaggedAddresses": {
					"lan": "10.0.0.55",
					"wan": "10.0.0.55"
				},
				"Meta": {
					"consul-network-segment": ""
				},
				"CreateIndex": 6,
				"ModifyIndex": 7
			}
		]`)
	})

	mux.HandleFunc("/v1/catalog/services", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"consul": [],
			"vault": [
				"active"
			]
		}`)
	})

	mux.HandleFunc("/v1/health/service/vault", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"Node": {
					"ID": "91011",
					"Node": "vault-dev-0",
					"Address": "10.0.0.55",
					"Datacenter": "dev",
					"TaggedAddresses": {
						"lan": "10.0.0.55",
						"wan": "10.0.0.55"
					},
					"Meta": {
						"consul-network-segment": ""
					},
					"CreateIndex": 20183,
					"ModifyIndex": 20185
				},
				"Service": {
					"ID": "vault:vault-dev-0.consul.localnet:8200",
					"Service": "vault",
					"Tags": [
						"active"
					],
					"Address": "vault-dev-0.consul.localnet",
					"Meta": null,
					"Port": 8200,
					"EnableTagOverride": false,
					"ProxyDestination": "",
					"Connect": {
						"Native": false,
						"Proxy": null
					},
					"CreateIndex": 20259,
					"ModifyIndex": 20328
				},
				"Checks": [
					{
						"Node": "vault-dev-0",
						"CheckID": "serfHealth",
						"Name": "Serf Health Status",
						"Status": "warning",
						"Notes": "",
						"Output": "Agent alive and reachable",
						"ServiceID": "",
						"ServiceName": "",
						"ServiceTags": [],
						"Definition": {},
						"CreateIndex": 20183,
						"ModifyIndex": 20183
					},
					{
						"Node": "vault-dev-0",
						"CheckID": "vault:vault-dev-0.consul.localnet:8200:vault-sealed-check",
						"Name": "Vault Sealed Status",
						"Status": "passing",
						"Notes": "Vault service is healthy when Vault is in an unsealed status and can become an active Vault server",
						"Output": "Vault Unsealed",
						"ServiceID": "vault:vault-dev-0.consul.localnet:8200",
						"ServiceName": "vault",
						"ServiceTags": [
							"active"
						],
						"Definition": {},
						"CreateIndex": 20260,
						"ModifyIndex": 20329
					}
				]
			}
		]`)
	})

	mux.HandleFunc("/v1/health/service/consul", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"Node": {
					"ID": "fbfe7e9b-5d30-284b-cc05-d2d5cc43688d",
					"Node": "consul-dev-0",
					"Address": "10.0.0.140",
					"Datacenter": "dev",
					"TaggedAddresses": {
						"lan": "10.0.0.140",
						"wan": "10.0.0.140"
					},
					"Meta": {
						"consul-network-segment": ""
					},
					"CreateIndex": 8,
					"ModifyIndex": 9
				},
				"Service": {
					"ID": "consul",
					"Service": "consul",
					"Tags": [],
					"Address": "",
					"Meta": null,
					"Port": 8300,
					"EnableTagOverride": false,
					"ProxyDestination": "",
					"Connect": {
						"Native": false,
						"Proxy": null
					},
					"CreateIndex": 8,
					"ModifyIndex": 8
				},
				"Checks": [
					{
						"Node": "consul-dev-0",
						"CheckID": "serfHealth",
						"Name": "Serf Health Status",
						"Status": "critical",
						"Notes": "",
						"Output": "Agent alive and reachable",
						"ServiceID": "",
						"ServiceName": "",
						"ServiceTags": [],
						"Definition": {},
						"CreateIndex": 8,
						"ModifyIndex": 8
					}
				]
			},
			{
				"Node": {
					"ID": "c7f88fba-f8d9-94a9-3627-523398acf7db",
					"Node": "consul-dev-1",
					"Address": "10.0.0.142",
					"Datacenter": "dev",
					"TaggedAddresses": {
						"lan": "10.0.0.142",
						"wan": "10.0.0.142"
					},
					"Meta": {
						"consul-network-segment": ""
					},
					"CreateIndex": 65,
					"ModifyIndex": 67
				},
				"Service": {
					"ID": "consul",
					"Service": "consul",
					"Tags": [],
					"Address": "",
					"Meta": null,
					"Port": 8300,
					"EnableTagOverride": false,
					"ProxyDestination": "",
					"Connect": {
						"Native": false,
						"Proxy": null
					},
					"CreateIndex": 65,
					"ModifyIndex": 65
				},
				"Checks": [
					{
						"Node": "consul-dev-1",
						"CheckID": "serfHealth",
						"Name": "Serf Health Status",
						"Status": "passing",
						"Notes": "",
						"Output": "Agent alive and reachable",
						"ServiceID": "",
						"ServiceName": "",
						"ServiceTags": [],
						"Definition": {},
						"CreateIndex": 65,
						"ModifyIndex": 65
					}
				]
			}
		]`)
	})
}

func Test_Datacenter_CollectMetrics_MeshCAExpiry(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	mux.HandleFunc("/v1/agent/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"Gauges": [
				{
					"Name": "consul.ip-10-10-50-192.mesh.active-root-ca.expiry",
					"Value": 3110400,
					"Labels": {}
				},
				{
					"Name": "consul.ip-10-10-50-192.mesh.active-signing-ca.expiry",
					"Value": 259200,
					"Labels": {}
				}
			],
			"Counters": [],
			"Samples": []
		}`)
	})
	mux.HandleFunc("/v1/catalog/nodes", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[]`)
	})
	mux.HandleFunc("/v1/catalog/services", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})

	dc := newTestDatacenter(t, mux, hostname, port)
	dc.CollectMetrics(false)

	result := dc.entity.Metrics[0].Metrics
	if got := result["mesh.activeRootCAExpiryInSeconds"]; got != float64(3110400) {
		t.Errorf("Expected mesh.activeRootCAExpiryInSeconds to be 3110400, got %v", got)
	}
	if got := result["mesh.activeSigningCAExpiryInSeconds"]; got != float64(259200) {
		t.Errorf("Expected mesh.activeSigningCAExpiryInSeconds to be 259200, got %v", got)
	}
}
