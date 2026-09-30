// Package datacenter handles grabbing of Datacenter level metrics via the leader agent
package datacenter

import (
	"errors"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-consul/src/agent"
	"github.com/newrelic/nri-consul/src/metrics"
)

// Datacenter represents the Datacenter
// Wraps the leader agent and Datacenter entity
type Datacenter struct {
	entity *integration.Entity
	leader *agent.Agent
}

// NewDatacenter creates a new datacenter wrapped around the leader Agent
func NewDatacenter(leader *agent.Agent, i *integration.Integration) (*Datacenter, error) {
	if leader == nil {
		return nil, errors.New("leader must not be nil")
	}

	dcName, err := getDatacenterName(leader.Client)
	if err != nil {
		return nil, err
	}

	dcEntity, err := i.Entity(*dcName, "co-datacenter")
	if err != nil {
		return nil, err
	}

	return &Datacenter{
		entity: dcEntity,
		leader: leader,
	}, nil
}

// getDatacenterName retrieves the Datacenter name from the leader
func getDatacenterName(client *api.Client) (*string, error) {
	self, err := client.Agent().Self()
	if err != nil {
		return nil, err
	}

	config, ok := self["Config"]
	if !ok {
		return nil, errors.New("no Config found")
	}

	dc, ok := config["Datacenter"]
	if !ok {
		return nil, errors.New("datacenter name not found")
	}

	dcName := dc.(string)
	return &dcName, nil
}

// CollectMetrics collects all datacenter level metrics. When collectHealthChecks
// is true, it also emits a ConsulHealthCheckSample per registered health check.
func (dc *Datacenter) CollectMetrics(collectHealthChecks bool) {
	metricSet := dc.entity.NewMetricSet("ConsulDatacenterSample",
		attribute.Attribute{Key: "displayName", Value: dc.entity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: dc.entity.Metadata.Namespace + ":" + dc.entity.Metadata.Name},
		attribute.Attribute{Key: "leader", Value: dc.leader.HostPort()},
	)

	// collect leader agent metrics
	if err := dc.leader.CollectCoreMetrics(metricSet, gaugeMetrics, counterMetrics, timerMetrics); err != nil {
		log.Error("Error collecting leader metrics for Datacenter: %s", err.Error())
	}

	// collect node count
	if err := dc.setNodeCountMetric(metricSet); err != nil {
		log.Error("Error collecting node count: %s", err.Error())
	}

	// collect node health counts
	if err := dc.collectStatusCounts(metricSet); err != nil {
		log.Error("Error getting node health counts: %s", err.Error())
	}

	if collectHealthChecks {
		if err := dc.collectHealthChecks(); err != nil {
			log.Error("Error collecting health checks for Datacenter: %s", err.Error())
		}
	}
}

func (dc *Datacenter) setNodeCountMetric(metricSet *metric.Set) error {
	nodes, _, err := dc.leader.Client.Catalog().Nodes(nil)
	if err != nil {
		return err
	}

	metrics.SetMetric(metricSet, "catalog.registeredNodes", len(nodes), metric.GAUGE)
	return nil
}

// collectStatusCounts aggregates health status across services on a node.
func (dc *Datacenter) collectStatusCounts(metricSet *metric.Set) error {
	services, _, err := dc.leader.Client.Catalog().Services(nil)
	if err != nil {
		return err
	}

	// keeps track of counts
	nodeCounts := map[string]int{
		"critical": 0,
		"up":       0,
		"warning":  0,
		"passing":  0,
	}

	// for each service look at the nodes that host it and count health
	for service := range services {
		entries, _, err := dc.leader.Client.Health().Service(service, "", false, nil)
		if err != nil {
			log.Error("Error getting nodes for service: %s", service)
			return err
		}

		for _, entry := range entries {
			switch entry.Checks.AggregatedStatus() {
			case api.HealthCritical:
				nodeCounts["critical"]++
			case api.HealthWarning:
				nodeCounts["warning"]++
			case api.HealthPassing:
				nodeCounts["up"]++
				nodeCounts["passing"]++
			}
		}
	}

	for status, count := range nodeCounts {
		metrics.SetMetric(metricSet, fmt.Sprintf("catalog.%sNodes", status), count, metric.GAUGE)
	}

	return nil
}
