package agent

import (
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/nri-consul/src/metrics"
)

const (
	leafCertExpiryAPIKey         = "consul.leaf-certs.cert_expiry"
	leafCertRenewalFailureAPIKey = "consul.leaf-certs.cert_renewal_failure"
)

// collectLeafCertMetrics emits one ConsulCertSample per labeled leaf-certificate
// gauge this agent is tracking (Consul 1.22+/2.0+, GH-23147). Unlike the scalar
// gauge/counter/timer definitions, these are per-service (labeled by datacenter,
// partition, namespace, service, kind), so a single named gauge can have multiple
// samples that need their own events rather than a single scalar value.
func (a *Agent) collectLeafCertMetrics() error {
	metricInfo, err := a.Client.Agent().Metrics()
	if err != nil {
		return err
	}

	for _, gauge := range metricInfo.Gauges {
		switch {
		case gaugeNameMatches(gauge.Name, leafCertExpiryAPIKey):
			a.emitCertSample(gauge.Labels, "cert.expiryInSeconds", gauge.Value)
		case gaugeNameMatches(gauge.Name, leafCertRenewalFailureAPIKey):
			a.emitCertSample(gauge.Labels, "cert.renewalFailing", gauge.Value)
		}
	}

	return nil
}

func (a *Agent) emitCertSample(labels map[string]string, valueName string, value float32) {
	metricSet := a.entity.NewMetricSet("ConsulCertSample",
		attribute.Attribute{Key: "displayName", Value: a.entity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: a.entity.Metadata.Namespace + ":" + a.entity.Metadata.Name},
		attribute.Attribute{Key: "datacenter", Value: labels["datacenter"]},
		attribute.Attribute{Key: "partition", Value: labels["partition"]},
		attribute.Attribute{Key: "namespace", Value: labels["namespace"]},
		attribute.Attribute{Key: "service", Value: labels["service"]},
		attribute.Attribute{Key: "kind", Value: labels["kind"]},
	)

	metrics.SetMetric(metricSet, valueName, value, metric.GAUGE)
}
