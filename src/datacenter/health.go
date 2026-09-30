package datacenter

import (
	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/nri-consul/src/metrics"
)

// healthStatusCode maps a Consul check status to a numeric code so it can be
// charted and alerted on (e.g. `max(check.statusCode) > 0`), the same way status
// is exposed as a service check elsewhere.
var healthStatusCode = map[string]int{
	api.HealthPassing:  0,
	api.HealthWarning:  1,
	api.HealthCritical: 2,
	api.HealthMaint:    3,
}

// collectHealthChecks emits one ConsulHealthCheckSample per registered health
// check (node and service level) in the datacenter, giving per-check status
// visibility instead of only the aggregate counts from collectStatusCounts.
func (dc *Datacenter) collectHealthChecks() error {
	checks, _, err := dc.leader.Client.Health().State(api.HealthAny, nil)
	if err != nil {
		return err
	}

	for _, check := range checks {
		metricSet := dc.entity.NewMetricSet("ConsulHealthCheckSample",
			attribute.Attribute{Key: "displayName", Value: dc.entity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: dc.entity.Metadata.Namespace + ":" + dc.entity.Metadata.Name},
			attribute.Attribute{Key: "node", Value: check.Node},
			attribute.Attribute{Key: "checkID", Value: check.CheckID},
			attribute.Attribute{Key: "checkName", Value: check.Name},
			attribute.Attribute{Key: "serviceID", Value: check.ServiceID},
			attribute.Attribute{Key: "serviceName", Value: check.ServiceName},
		)

		metrics.SetMetric(metricSet, "check.status", check.Status, metric.ATTRIBUTE)
		metrics.SetMetric(metricSet, "check.output", check.Output, metric.ATTRIBUTE)
		metrics.SetMetric(metricSet, "check.statusCode", statusCode(check.Status), metric.GAUGE)
	}

	return nil
}

// statusCode converts a Consul check status string to its numeric code, or -1 if unknown.
func statusCode(status string) int {
	if code, ok := healthStatusCode[status]; ok {
		return code
	}
	return -1
}
