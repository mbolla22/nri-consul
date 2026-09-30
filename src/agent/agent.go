// Package agent handles Agent entity creation and inventory/metric collection
package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-consul/src/args"
	"github.com/newrelic/nri-consul/src/metrics"
)

// number of workers there can be per pool
const workerCount = 5

// Agent represents a Consul agent.
// It's comprised of the client connected to that agent
// and the Entity representing it.
type Agent struct {
	entity     *integration.Entity
	Client     *api.Client
	datacenter string
	ipAddr     string
	name       string
}

// CreateAgents creates an Agent structure for every Agent member of the LAN cluster
func CreateAgents(client *api.Client, i *integration.Integration, args *args.ArgumentList) (agents []*Agent, leader *Agent, err error) {
	members, err := client.Agent().Members(false)
	if err != nil {
		log.Error("Error getting members: %s", err.Error())
		return
	}

	leaderAddr, err := getLeaderAddr(client)
	if err != nil {
		log.Error("Error getting leader address: %s", err.Error())
		return
	}

	agents = make([]*Agent, 0, len(members))
	for _, member := range members {

		memberNameIDAttr := integration.NewIDAttribute("co-agent", member.Name)
		entity, err := i.Entity(fmt.Sprintf("%s:%d", member.Addr, member.Port), "co-agent", memberNameIDAttr)
		if err != nil {
			log.Error("Error creating entity for Agent '%s': %s", member.Name, err.Error())
			continue
		}

		apiConfig, err := args.CreateAPIConfig(member.Addr)
		if err != nil {
			log.Error("Error creating httpClient for Agent '%s': %s", member.Name, err.Error())
			continue
		}

		client, err = api.NewClient(apiConfig)
		if err != nil {
			log.Error("Error creating client for Agent '%s': %s", member.Name, err.Error())
			continue
		}

		agent := NewAgent(client, entity, member.Name, member.Addr, member.Tags["dc"])
		agents = append(agents, agent)

		// we need to identify the leader to collect catalog
		if member.Addr == leaderAddr {
			leader = agent
		}
	}

	err = nil

	return
}

// NewAgent creates a new agent from the given client and Entity
func NewAgent(client *api.Client, entity *integration.Entity, name, ipAddr, datacenter string) *Agent {
	return &Agent{
		Client:     client,
		entity:     entity,
		ipAddr:     ipAddr,
		datacenter: datacenter,
		name:       name,
	}
}

func (a *Agent) processConfig(config map[string]interface{}, configPrefix string) {
	for key, value := range config {
		switch v := value.(type) {
		case map[string]interface{}:
			log.Debug("Not processing config param '%s' nested object", key)
		case string:
			if v != "" {
				a.setInventoryItem(configPrefix+"/"+key, "value", v)
			}
		case []interface{}:
			if len(v) > 0 {
				if stringVal, err := arrayToString(v); err != nil {
					log.Debug("Unable to store config param '%s': %s", key, err.Error())
				} else {
					a.setInventoryItem(configPrefix+"/"+key, "value", *stringVal)
				}
			}
		default:
			a.setInventoryItem(configPrefix+"/"+key, "value", v)
		}
	}
}

// setInventoryItem adds a wrapper around setting an inventory item
func (a *Agent) setInventoryItem(key, field string, value interface{}) {
	if err := a.entity.SetInventoryItem(key, field, value); err != nil {
		log.Debug("Error setting Inventory item '%s' on Agent '%s': %s", key, a.entity.Metadata.Name)
	}
}

// collectPeerCount counts the number of peers for an agent.
func (a *Agent) collectPeerCount(metricSet *metric.Set) error {
	log.Debug("Starting peer count collection for Agent %s", a.entity.Metadata.Name)

	peers, err := a.Client.Status().Peers()
	if err != nil {
		return err
	}

	metrics.SetMetric(metricSet, "agent.peers", len(peers), metric.GAUGE)

	log.Debug("Finished peer count collection for Agent %s", a.entity.Metadata.Name)
	return nil
}

func (a *Agent) collectLatencyMetrics(metricSet *metric.Set) error {
	log.Debug("Starting latency metric collection for Agent %s", a.entity.Metadata.Name)

	nodes, _, err := a.Client.Coordinate().Nodes(nil)
	if err != nil {
		return err
	}

	if len(nodes) == 1 {
		return errors.New("could not collect latency metrics because the cluster only contains 1 node")
	}

	agentNode := findNode(a.name, nodes)
	if agentNode == nil {
		return fmt.Errorf("could not collect latency metrics because the node with name %s could not be found be found in the list of nodes: %v", a.name, nodes)
	}

	// calculate and populate metrics
	calculateLatencyMetrics(metricSet, agentNode, nodes)

	log.Debug("Finished latency metric collection for Agent %s", a.entity.Metadata.Name)
	return nil
}

// HostPort returns the host:port of the agent
func (a *Agent) HostPort() string {
	return a.entity.Metadata.Name
}

// CollectCoreMetrics collects metrics for an Agent
func (a *Agent) CollectCoreMetrics(metricSet *metric.Set, gaugeDefs, counterDefs []*metrics.MetricDefinition, timerDefs []*metrics.TimerDefinition) error {
	log.Debug("Starting core metric collection for Agent %s", a.entity.Metadata.Name)
	metricInfo, err := a.Client.Agent().Metrics()
	if err != nil {
		return err
	}

	// collect gauges
	if gaugeDefs != nil {
		collectGaugeMetrics(metricSet, metricInfo.Gauges, gaugeDefs)
	}

	// collect counters
	if counterDefs != nil {
		collectCounterMetrics(metricSet, metricInfo.Counters, counterDefs)
	}

	// collect timers
	if timerDefs != nil {
		collectTimerMetrics(metricSet, metricInfo.Samples, timerDefs)
	}

	log.Debug("Finished core metric collection for Agent %s", a.entity.Metadata.Name)
	return nil
}

// gaugeNameMatches compares a raw gauge name against a configured API key.
// Consul prefixes gauge names (but not counters/timers) with the agent's
// hostname when telemetry.disable_hostname is false (the default), e.g.
// "consul.<hostname>.runtime.num_goroutines" instead of "consul.runtime.num_goroutines".
// This matches either form.
func gaugeNameMatches(name, apiKey string) bool {
	if name == apiKey {
		return true
	}

	suffix := strings.TrimPrefix(apiKey, "consul.")
	return strings.HasPrefix(name, "consul.") && strings.HasSuffix(name, "."+suffix)
}

func collectGaugeMetrics(metricSet *metric.Set, gauges []api.GaugeValue, defs []*metrics.MetricDefinition) {
	for _, def := range defs {
		found := false

		// Look through all gauges for metric
		for _, gauge := range gauges {
			// If found, record and break
			if gaugeNameMatches(gauge.Name, def.APIKey) {
				value := gauge.Value

				// special case where we need to convert nanoseconds to milliseconds
				if def.APIKey == "consul.runtime.total_gc_pause_ns" {
					value /= 1000000
				}

				found = true
				metrics.SetMetric(metricSet, def.MetricName, value, def.SourceType)
				break
			}
		}

		if !found {
			log.Debug("Did not find metric '%s' matching API key '%s'", def.MetricName, def.APIKey)
		}
	}
}

func collectCounterMetrics(metricSet *metric.Set, counters []api.SampledValue, defs []*metrics.MetricDefinition) {
	for _, def := range defs {
		found := false

		// Look through all counters for metric
		for _, counter := range counters {
			// If found, record and break
			if def.APIKey == counter.Name {
				found = true
				metrics.SetMetric(metricSet, def.MetricName, counter.Count, def.SourceType)
				break
			}
		}

		if !found {
			log.Debug("Did not find metric '%s' matching API key '%s'", def.MetricName, def.APIKey)
		}
	}
}

func collectTimerMetrics(metricSet *metric.Set, timers []api.SampledValue, defs []*metrics.TimerDefinition) {
	lookup := make(map[string]*api.SampledValue)

	for _, def := range defs {

		// Check if the timer is cached, if not search for it.
		sample, ok := lookup[def.APIKey]
		if !ok {
			for _, timer := range timers {
				if def.APIKey == timer.Name {
					sample = &timer
					lookup[def.APIKey] = sample
					break
				}
			}
		}

		if sample == nil {
			log.Debug("Did not find metric '%s' matching API key '%s'", def.MetricName, def.APIKey)
			continue
		}

		// Calculate/collect statistical sample
		value := calculateStatValue(def.Operation, sample)
		metrics.SetMetric(metricSet, def.MetricName, value, def.SourceType)
	}
}

func calculateStatValue(operation metrics.StatOperation, sample *api.SampledValue) float64 {
	var value float64
	switch operation {
	case metrics.Average:
		value = sample.Mean
	case metrics.Max:
		value = sample.Max
	case metrics.Count:
		value = float64(sample.Count)
	}

	return value
}

// arrayToString converts an interface array to a comma delimited string if possible
func arrayToString(input []interface{}) (*string, error) {
	stringElements := make([]string, len(input))

	for i, elem := range input {
		elemString, ok := elem.(string)
		if !ok {
			return nil, fmt.Errorf("could not convert %v of type %T to string", elem, elem)
		}

		stringElements[i] = elemString
	}

	outString := strings.Join(stringElements, ",")

	return &outString, nil
}

func getLeaderAddr(client *api.Client) (string, error) {
	leaderAddr, err := client.Status().Leader()
	if err != nil {
		return "", err
	}

	// Addr comes in the form IP:Port splitting and only returning IP
	return strings.Split(leaderAddr, ":")[0], nil
}
