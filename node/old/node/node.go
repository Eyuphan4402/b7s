package node

import (
	"sync"

	"github.com/armon/go-metrics"
	"github.com/rs/zerolog"

	"github.com/blocklessnetwork/b7s-attributes/attributes"
	"github.com/blocklessnetwork/b7s/host"
	"github.com/blocklessnetwork/b7s/models/blockless"
	"github.com/blocklessnetwork/b7s/models/execute"
	"github.com/blocklessnetwork/b7s/models/response"
	"github.com/blocklessnetwork/b7s/node/internal/waitmap"
	"github.com/blocklessnetwork/b7s/telemetry/tracing"
)

// Node is the entity that actually provides the main Blockless node functionality.
// It listens for messages coming from the wire and processes them. Depending on the
// node role, which is determined on construction, it may process messages in different ways.
// For example, upon receiving a message requesting execution of a Blockless function,
// a Worker Node will use the `Execute` component to fulfill the execution request.
// On the other hand, a Head Node will issue a roll call and eventually
// delegate the execution to the chosend Worker Node.
type Node struct {
	cfg Config

	log      zerolog.Logger
	host     *host.Host
	executor blockless.Executor
	fstore   FStore

	sema       chan struct{}
	wg         *sync.WaitGroup
	subgroups  workSubgroups
	attributes *attributes.Attestation

	rollCall *rollCallQueue

	// clusters maps request ID to the cluster the node belongs to.
	clusters map[string]consensusExecutor

	// clusterLock is used to synchronize access to the `clusters` map.
	clusterLock sync.RWMutex

	executeResponses   *waitmap.WaitMap[string, execute.ResultMap]
	consensusResponses *waitmap.WaitMap[string, response.FormCluster]

	// Telemetry
	tracer  *tracing.Tracer
	metrics *metrics.Metrics
}
