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

func newTestDatacenter(t *testing.T, mux *http.ServeMux, hostname, port string) *Datacenter {
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

	return &Datacenter{
		entity: dcEntity,
		leader: agent.NewAgent(client, agentEntity, "", "", ""),
	}
}

func Test_CollectHealthChecks_Success(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	mux.HandleFunc("/v1/health/state/any", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"Node": "consul-dev-0",
				"CheckID": "serfHealth",
				"Name": "Serf Health Status",
				"Status": "critical",
				"Notes": "",
				"Output": "Agent not reachable",
				"ServiceID": "",
				"ServiceName": "",
				"ServiceTags": []
			},
			{
				"Node": "vault-dev-0",
				"CheckID": "vault:vault-dev-0.consul.localnet:8200:vault-sealed-check",
				"Name": "Vault Sealed Status",
				"Status": "passing",
				"Notes": "",
				"Output": "Vault Unsealed",
				"ServiceID": "vault:vault-dev-0.consul.localnet:8200",
				"ServiceName": "vault",
				"ServiceTags": ["active"]
			}
		]`)
	})

	dc := newTestDatacenter(t, mux, hostname, port)

	if err := dc.collectHealthChecks(); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	if len(dc.entity.Metrics) != 2 {
		t.Fatalf("Expected 2 metric sets, got %d", len(dc.entity.Metrics))
	}

	expectedFirst := map[string]interface{}{
		"event_type":       "ConsulHealthCheckSample",
		"displayName":      dc.entity.Metadata.Name,
		"entityName":       dc.entity.Metadata.Namespace + ":" + dc.entity.Metadata.Name,
		"node":             "consul-dev-0",
		"checkID":          "serfHealth",
		"checkName":        "Serf Health Status",
		"serviceID":        "",
		"serviceName":      "",
		"check.status":     "critical",
		"check.output":     "Agent not reachable",
		"check.statusCode": float64(2),
	}

	if result := dc.entity.Metrics[0].Metrics; !reflect.DeepEqual(result, expectedFirst) {
		t.Errorf("Expected %+v got %+v", expectedFirst, result)
	}

	expectedSecond := map[string]interface{}{
		"event_type":       "ConsulHealthCheckSample",
		"displayName":      dc.entity.Metadata.Name,
		"entityName":       dc.entity.Metadata.Namespace + ":" + dc.entity.Metadata.Name,
		"node":             "vault-dev-0",
		"checkID":          "vault:vault-dev-0.consul.localnet:8200:vault-sealed-check",
		"checkName":        "Vault Sealed Status",
		"serviceID":        "vault:vault-dev-0.consul.localnet:8200",
		"serviceName":      "vault",
		"check.status":     "passing",
		"check.output":     "Vault Unsealed",
		"check.statusCode": float64(0),
	}

	if result := dc.entity.Metrics[1].Metrics; !reflect.DeepEqual(result, expectedSecond) {
		t.Errorf("Expected %+v got %+v", expectedSecond, result)
	}
}

func Test_CollectHealthChecks_EndpointFails(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	mux.HandleFunc("/v1/health/state/any", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	dc := newTestDatacenter(t, mux, hostname, port)

	if err := dc.collectHealthChecks(); err == nil {
		t.Error("Expected error")
	}

	if len(dc.entity.Metrics) != 0 {
		t.Errorf("Expected no metric sets to be created, got %d", len(dc.entity.Metrics))
	}
}

func Test_StatusCode(t *testing.T) {
	tests := []struct {
		status   string
		expected int
	}{
		{api.HealthPassing, 0},
		{api.HealthWarning, 1},
		{api.HealthCritical, 2},
		{api.HealthMaint, 3},
		{"unknown", -1},
	}

	for _, test := range tests {
		if got := statusCode(test.status); got != test.expected {
			t.Errorf("statusCode(%q) = %d, expected %d", test.status, got, test.expected)
		}
	}
}

func Test_Datacenter_CollectMetrics_WithHealthChecks(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	setMetricMuxes(mux)
	mux.HandleFunc("/v1/health/state/any", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{
				"Node": "consul-dev-0",
				"CheckID": "serfHealth",
				"Name": "Serf Health Status",
				"Status": "critical",
				"Output": "Agent not reachable"
			}
		]`)
	})

	dc := newTestDatacenter(t, mux, hostname, port)

	dc.CollectMetrics(true)

	// index 0 is the ConsulDatacenterSample; the health check should be appended after it
	if len(dc.entity.Metrics) != 2 {
		t.Fatalf("Expected 2 metric sets, got %d", len(dc.entity.Metrics))
	}

	if eventType := dc.entity.Metrics[1].Metrics["event_type"]; eventType != "ConsulHealthCheckSample" {
		t.Errorf("Expected second metric set to be a ConsulHealthCheckSample, got %v", eventType)
	}
}

func Test_Datacenter_CollectMetrics_HealthChecksDisabled(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	setMetricMuxes(mux)
	mux.HandleFunc("/v1/health/state/any", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Should not have called /v1/health/state/any when health checks are disabled")
	})

	dc := newTestDatacenter(t, mux, hostname, port)

	dc.CollectMetrics(false)

	if len(dc.entity.Metrics) != 1 {
		t.Fatalf("Expected only the ConsulDatacenterSample metric set, got %d", len(dc.entity.Metrics))
	}
}
