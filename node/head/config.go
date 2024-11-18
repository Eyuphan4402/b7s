package head

import (
	"time"

	"github.com/blocklessnetwork/b7s/consensus"
	"github.com/blocklessnetwork/b7s/models/blockless"
)

// Option can be used to set Node configuration options.
type Option func(*Config)

// DefaultConfig represents the default settings for the node.
var DefaultConfig = Config{
	Topics:                  []string{blockless.DefaultTopic},
	HealthInterval:          blockless.DefaultHealthInterval,
	Concurrency:             blockless.DefaultConcurrency,
	RollCallTimeout:         DefaultRollCallTimeout,
	ExecutionTimeout:        DefaultExecutionTimeout,
	ClusterFormationTimeout: DefaultClusterFormationTimeout,
	DefaultConsensus:        DefaultConsensusAlgorithm,
}

// TODO: Head node does not need to subscribe to topics at all.

// Config represents the Node configuration.
type Config struct {
	Topics                  []string       // Topics to subscribe to.
	HealthInterval          time.Duration  // How often should we emit the health ping.
	Concurrency             uint           // How many requests should the node process in parallel.
	RollCallTimeout         time.Duration  // How long do we wait for roll call responses.
	ExecutionTimeout        time.Duration  // How long does the head node wait for worker nodes to send their execution results.
	ClusterFormationTimeout time.Duration  // How long do we wait for the nodes to form a cluster for an execution.
	DefaultConsensus        consensus.Type // Default consensus algorithm to use.
}

func (c Config) Valid() error {
	return nil
}