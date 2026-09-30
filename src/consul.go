package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-consul/src/agent"
	"github.com/newrelic/nri-consul/src/args"
	"github.com/newrelic/nri-consul/src/datacenter"
)

const (
	integrationName = "com.newrelic.consul"
)

var (
	integrationVersion = "0.0.0"
	gitCommit          = ""
	buildDate          = ""
)

func main() {
	var args args.ArgumentList
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	if args.ShowVersion {
		fmt.Printf(
			"New Relic %s integration Version: %s, Platform: %s, GoVersion: %s, GitCommit: %s, BuildDate: %s\n",
			strings.Title(strings.Replace(integrationName, "com.newrelic.", "", 1)),
			integrationVersion,
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			runtime.Version(),
			gitCommit,
			buildDate)
		os.Exit(0)
	}

	// Setup logging with verbose
	log.SetupLogging(args.Verbose)

	apiConfig, err := args.CreateAPIConfig(args.Hostname)
	if err != nil {
		log.Error("Error creating HTTP API client, please check configuration: %s", err.Error())
		os.Exit(1)
	}

	// create client
	client, err := api.NewClient(apiConfig)
	if err != nil {
		log.Error("Error creating API client, please check configuration: %s", err.Error())
		os.Exit(1)
	}

	var collectionError error
	if args.FanOut {
		collectionError = fanOutCollection(client, i, &args)
	} else {
		collectionError = localCollection(client, i, &args)
	}

	if collectionError != nil {
		log.Error("Error collecting metrics: %s", collectionError.Error())
		os.Exit(1)
	}

	if err = i.Publish(); err != nil {
		log.Error("Failed to publish metrics: %s", err.Error())
		os.Exit(1)
	}
}

func fanOutCollection(client *api.Client, i *integration.Integration, args *args.ArgumentList) error {
	// Create the list of agents in LAN pool
	agents, leader, err := agent.CreateAgents(client, i, args)
	if err != nil {
		return fmt.Errorf("Error creating Agent entities: %s", err.Error())
	}

	dc, err := datacenter.NewDatacenter(leader, i)
	if err != nil {
		log.Error("Error creating Datacenter entity: %s", err.Error())
	} else if args.HasMetrics() {
		dc.CollectMetrics(args.EnableHealthChecks)
	}

	// Collect inventory for agents
	if args.HasInventory() {
		agent.CollectInventory(agents)
	}

	// Collect metrics for Agents and cluster
	if args.HasMetrics() {
		agent.CollectMetrics(agents, args.EnableCertMetrics)
	}

	return nil
}

func maybeConvertBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	default:
		return false, fmt.Errorf("Unexpected type: %T", v)
	case bool:
		return value.(bool), nil
	case string:
		val, err := strconv.ParseBool(value.(string))
		if err != nil {
			return false, fmt.Errorf("Unable to convert %v to a bool: %v", value, err)
		}
		return val, nil
	}
}

func localCollection(client *api.Client, i *integration.Integration, args *args.ArgumentList) error {
	localAgentData, err := client.Agent().Self()
	if err != nil {
		return fmt.Errorf("Failed to collect local agent data: %v", err)
	}

	// TODO: It would be nice if this was available to us as a MemberAgent
	//       object but I don't think it is
	member, ok := localAgentData["Member"]
	if !ok {
		return fmt.Errorf("Failed to get local agent member: %v", ok)
	}

	memberName, ok := member["Name"].(string)
	if !ok {
		return fmt.Errorf("Failed to get member name: %v", ok)
	}

	memberAddr, ok := member["Addr"].(string)
	if !ok {
		return fmt.Errorf("Failed to get member address: %v", ok)
	}

	memberPort, ok := member["Port"]
	if !ok {
		return fmt.Errorf("Failed to get member port: %v", ok)
	}

	memberDataCenter, ok := member["Tags"].(map[string]interface{})["dc"].(string)
	if !ok {
		return fmt.Errorf("Failed to get member datacenter: %v", ok)
	}

	var isLeaderValue interface{}
	if args.CheckLeadership {
		isLeaderValue, ok = localAgentData["Stats"]["consul"].(map[string]interface{})["leader"]
		if !ok {
			isLeaderValue, ok = localAgentData["Stats"]["consul"].(map[string]interface{})["server"]
			if !ok {
				return fmt.Errorf("Failed to check for leadership: %v", ok)
			}
		}
	}

	// The key currently comes back to us as a string, so we convert it if we need to
	// Little bit of future proofing in case the api changes
	isLeader, err := maybeConvertBool(isLeaderValue)
	if err != nil {
		log.Error("Leadership value is not a bool. Defaulting to false: %v", err)
		isLeader = false
	}

	agentNameIDAttr := integration.NewIDAttribute("co-agent", memberName)
	entity, err := i.Entity(fmt.Sprintf("%s:%v", memberAddr, memberPort), "co-agent", agentNameIDAttr)
	if err != nil {
		return fmt.Errorf("failed to create newrelic entity: %v", err)
	}
	agentInstance := agent.NewAgent(client, entity, memberName, memberAddr, memberDataCenter)

	if args.HasMetrics() {
		if isLeader {
			log.Debug("Checking Leader Metrics")
			dc, err := datacenter.NewDatacenter(agentInstance, i)
			if err != nil {
				log.Error("Failed to get datacenter metrics: %v", err)
			} else {
				dc.CollectMetrics(args.EnableHealthChecks)
			}
		} else {
			log.Debug("Not Checking Leader Metrics")
		}
		agent.CollectMetricsFromOne(agentInstance, args.EnableCertMetrics)
	}

	if args.HasInventory() {
		agent.CollectInventoryFromOne(agentInstance)
	}

	return nil
}
