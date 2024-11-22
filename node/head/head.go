package head

import (
	"fmt"

	"github.com/armon/go-metrics"
	"github.com/google/uuid"

	"github.com/blocklessnetwork/b7s/info"
	"github.com/blocklessnetwork/b7s/models/execute"
	"github.com/blocklessnetwork/b7s/models/response"
	"github.com/blocklessnetwork/b7s/node/internal/node"
	"github.com/blocklessnetwork/b7s/node/internal/waitmap"
)

type HeadNode struct {
	node.Core

	cfg Config

	rollCall           *rollCallQueue
	consensusResponses *waitmap.WaitMap[string, response.FormCluster]
	executeResponses   *waitmap.WaitMap[string, execute.NodeResult]
}

func New(core node.Core, options ...Option) (*HeadNode, error) {

	// Initialize config.
	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	err := cfg.Valid()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// TODO: Tracing.

	head := &HeadNode{
		Core: core,
		cfg:  cfg,

		rollCall:           newQueue(rollCallQueueBufferSize),
		consensusResponses: waitmap.New[string, response.FormCluster](0),
		executeResponses:   waitmap.New[string, execute.NodeResult](executionResultCacheSize),
	}

	head.Metrics().SetGaugeWithLabels(node.NodeInfoMetric, 1,
		[]metrics.Label{
			{Name: "id", Value: head.ID()},
			{Name: "version", Value: info.VcsVersion()},
			{Name: "role", Value: "head"},
		})

	return head, nil
}

func newRequestID() string {
	return uuid.New().String()
}
