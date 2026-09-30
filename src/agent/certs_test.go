package agent

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-consul/src/args"
	"github.com/newrelic/nri-consul/src/testutils"
)

func newTestAgent(t *testing.T, mux *http.ServeMux, hostname, port string) *Agent {
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

	return &Agent{Client: client, entity: entity}
}

func Test_CollectLeafCertMetrics_Success(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	mux.HandleFunc("/v1/agent/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"Gauges": [
				{
					"Name": "consul.ip-10-10-50-192.leaf-certs.cert_expiry",
					"Value": 259200,
					"Labels": {"datacenter": "dc1", "partition": "default", "namespace": "default", "service": "web", "kind": "connect-proxy"}
				},
				{
					"Name": "consul.ip-10-10-50-192.leaf-certs.cert_expiry",
					"Value": 86400,
					"Labels": {"datacenter": "dc1", "partition": "default", "namespace": "default", "service": "payments", "kind": "connect-proxy"}
				},
				{
					"Name": "consul.ip-10-10-50-192.leaf-certs.cert_renewal_failure",
					"Value": 1,
					"Labels": {"datacenter": "dc1", "partition": "default", "namespace": "default", "service": "payments", "kind": "connect-proxy", "reason": "rate_limited"}
				},
				{
					"Name": "consul.ip-10-10-50-192.leaf-certs.entries_count",
					"Value": 2,
					"Labels": {}
				}
			]
		}`)
	})

	agent := newTestAgent(t, mux, hostname, port)

	if err := agent.collectLeafCertMetrics(); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	if len(agent.entity.Metrics) != 3 {
		t.Fatalf("Expected 3 ConsulCertSample events (entries_count is not per-service), got %d", len(agent.entity.Metrics))
	}

	expectedWeb := map[string]interface{}{
		"event_type":           "ConsulCertSample",
		"displayName":          agent.entity.Metadata.Name,
		"entityName":           agent.entity.Metadata.Namespace + ":" + agent.entity.Metadata.Name,
		"datacenter":           "dc1",
		"partition":            "default",
		"namespace":            "default",
		"service":              "web",
		"kind":                 "connect-proxy",
		"cert.expiryInSeconds": float64(259200),
	}
	if got := agent.entity.Metrics[0].Metrics; got["service"] != expectedWeb["service"] || got["cert.expiryInSeconds"] != expectedWeb["cert.expiryInSeconds"] {
		t.Errorf("Expected %+v got %+v", expectedWeb, got)
	}

	if got := agent.entity.Metrics[1].Metrics["cert.expiryInSeconds"]; got != float64(86400) {
		t.Errorf("Expected payments cert.expiryInSeconds to be 86400, got %v", got)
	}

	if got := agent.entity.Metrics[2].Metrics["cert.renewalFailing"]; got != float64(1) {
		t.Errorf("Expected payments cert.renewalFailing to be 1, got %v", got)
	}
}

func Test_CollectLeafCertMetrics_EndpointFails(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	mux.HandleFunc("/v1/agent/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	agent := newTestAgent(t, mux, hostname, port)

	if err := agent.collectLeafCertMetrics(); err == nil {
		t.Error("Expected error")
	}

	if len(agent.entity.Metrics) != 0 {
		t.Errorf("Expected no metric sets to be created, got %d", len(agent.entity.Metrics))
	}
}

func Test_CollectLeafCertMetrics_NoLeafCertGauges(t *testing.T) {
	mux, hostname, port, serverClose := testutils.SetupServer()
	defer serverClose()

	mux.HandleFunc("/v1/agent/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Gauges": [{"Name": "consul.runtime.num_goroutines", "Value": 10, "Labels": {}}]}`)
	})

	agent := newTestAgent(t, mux, hostname, port)

	if err := agent.collectLeafCertMetrics(); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	if len(agent.entity.Metrics) != 0 {
		t.Errorf("Expected no ConsulCertSample events when no leaf-cert gauges are present, got %d", len(agent.entity.Metrics))
	}
}
