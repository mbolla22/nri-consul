// Package args contains the argument list, defined as a struct, along with a method that validates passed-in args
package args

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	goCleanhttp "github.com/hashicorp/go-cleanhttp"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/v3/args"
)

// ArgumentList struct that holds all Consul arguments
type ArgumentList struct {
	sdkArgs.DefaultArgumentList
	Hostname               string `default:"localhost" help:"The agent node Hostname or IP address to connect to"`
	Port                   string `default:"8500" help:"Port to connect to agent node"`
	Token                  string `default:"" help:"ACL Token if token authentication is enabled"`
	Timeout                string `default:"0s" help:"Timeout for an API call"`
	EnableSSL              bool   `default:"false" help:"If true will use SSL encryption, false will not use encryption"`
	TrustServerCertificate bool   `default:"false" help:"If true server certificate is not verified for SSL. If false certificate will be verified against supplied certificate"`
	CABundleFile           string `default:"" help:"Alternative Certificate Authority bundle file"`
	CABundleDir            string `default:"" help:"Alternative Certificate Authority bundle directory"`
	FanOut                 bool   `default:"true" help:"If true will attempt to gather metrics from all other nodes in consul cluster"`
	CheckLeadership        bool   `default:"true" help:"Check leadership on consul server. This should be disabled on consul in client mode"`
	EnableHealthChecks     bool   `default:"true" help:"If true, collect a ConsulHealthCheckSample event per registered health check (node and service level), reporting its status. This should be disabled on large clusters with a lot of checks to avoid excessive event cardinality"`
	EnableCertMetrics      bool   `default:"true" help:"If true, collect a ConsulCertSample event per tracked leaf certificate, reporting expiry and renewal-failure status (Consul 1.22+/2.0+). Also enables agent TLS/mesh CA expiry gauges when present."`
	ShowVersion            bool   `default:"false" help:"Print build information and exit"`
}

// Validate validates Consul arguments
func (al ArgumentList) Validate() error {
	if al.EnableSSL {
		if !al.TrustServerCertificate && al.CABundleDir == "" && al.CABundleFile == "" {
			return errors.New("invalid configuration: must specify a certificate file or bundle when using SSL and not trusting server certificate")
		}
	}

	return nil
}

// CreateAPIConfig creates an API config from the argument list
func (al ArgumentList) CreateAPIConfig(hostname string) (*api.Config, error) {
	// Since we are creating the HttpClient instead of using the default (so we can define a Timeout)
	// we must set the Transport with the same defaults used in consul's api.NewClient.
	config := &api.Config{
		Address: fmt.Sprintf("%s:%s", hostname, al.Port),
		Token:   al.Token,
		Scheme:  "http",
		// Using the same as the consul api.NewClient
		Transport: goCleanhttp.DefaultPooledTransport(),
	}

	// setup SSL if enabled
	if al.EnableSSL {
		config.TLSConfig = api.TLSConfig{
			CAFile:             al.CABundleFile,
			CAPath:             al.CABundleDir,
			InsecureSkipVerify: al.TrustServerCertificate,
		}
		config.Scheme = "https"
	}

	httpClient, err := api.NewHttpClient(config.Transport, config.TLSConfig)
	if err != nil {
		return nil, err
	}

	duration, err := time.ParseDuration(al.Timeout)
	if err != nil {
		return nil, err
	}

	httpClient.Timeout = duration
	config.HttpClient = httpClient

	return config, nil
}
