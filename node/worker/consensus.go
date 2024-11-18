package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/blocklessnetwork/b7s/consensus"
	"github.com/blocklessnetwork/b7s/consensus/pbft"
	"github.com/blocklessnetwork/b7s/consensus/raft"
	"github.com/blocklessnetwork/b7s/models/codes"
	"github.com/blocklessnetwork/b7s/models/execute"
	"github.com/blocklessnetwork/b7s/models/request"
	"github.com/blocklessnetwork/b7s/models/response"
	"github.com/blocklessnetwork/b7s/telemetry/tracing"
)

// consensusExecutor defines the interface we have for managing clustered execution.
// Execute often does not mean a direct execution but instead just pipelining the request, where execution is done asynchronously.
type consensusExecutor interface {
	Consensus() consensus.Type
	Execute(from peer.ID, id string, timestamp time.Time, request execute.Request) (codes.Code, execute.Result, error)
	Shutdown() error
}

func (w *Worker) createRaftCluster(ctx context.Context, from peer.ID, fc request.FormCluster) error {

	// Add a callback function to send the execution result to origin.
	sendFn := func(req raft.FSMLogEntry, res execute.NodeResult) {

		ctx, cancel := context.WithTimeout(context.Background(), consensusClusterSendTimeout)
		defer cancel()

		metadata, err := w.cfg.MetadataProvider.Metadata(req.Execute, res.Result.Result)
		if err != nil {
			w.Log().Warn().Err(err).Msg("could not get metadata")
		}
		res.Metadata = metadata

		// TODO: Think: response.WorkOrder vs execute.Result vs execute.NodeResult => which one makes the most sense where
		msg := response.WorkOrder{
			Code:      res.Code,
			RequestID: fc.RequestID,
			Result:    res,
		}

		err = w.Send(ctx, req.Origin, &msg)
		if err != nil {
			w.Log().Error().Err(err).Stringer("peer", req.Origin).Msg("could not send execution result to node")
		}
	}

	// Add a callback function to cache the execution result
	cacheFn := func(req raft.FSMLogEntry, res execute.NodeResult) {
		w.executeResponses.Set(req.RequestID, res)
	}

	rh, err := raft.New(
		*w.Log(),
		w.Host(),
		w.cfg.Workspace,
		fc.RequestID,
		w.executor,
		fc.Peers,
		raft.WithCallbacks(cacheFn, sendFn),
	)
	if err != nil {
		return fmt.Errorf("could not create raft node: %w", err)
	}

	w.clusters.Set(fc.RequestID, rh)

	err = w.Send(ctx, from, fc.Response(codes.OK).WithConsensus(fc.Consensus))
	if err != nil {
		return fmt.Errorf("could not send cluster confirmation message: %w", err)
	}

	return nil
}

func (w *Worker) createPBFTCluster(ctx context.Context, from peer.ID, fc request.FormCluster) error {

	cacheFn := func(requestID string, origin peer.ID, req execute.Request, res execute.NodeResult) {
		w.executeResponses.Set(fc.RequestID, res)
	}

	// If we have tracing enabled we will have trace info in the context.
	// If not, there might be trace info in the message so just use that.
	ti := tracing.GetTraceInfo(ctx)
	if ti.Empty() {
		ti = fc.TraceInfo
	}

	ph, err := pbft.NewReplica(
		*w.Log(),
		w.Host(),
		w.executor,
		fc.Peers,
		fc.RequestID,
		pbft.WithPostProcessors(cacheFn),
		pbft.WithTraceInfo(ti),
		pbft.WithMetadataProvider(w.cfg.MetadataProvider),
	)
	if err != nil {
		return fmt.Errorf("could not create PBFT node: %w", err)
	}

	w.clusters.Set(fc.RequestID, ph)

	err = w.Send(ctx, from, fc.Response(codes.OK).WithConsensus(fc.Consensus))
	if err != nil {
		return fmt.Errorf("could not send cluster confirmation message: %w", err)
	}

	return nil
}

func (w *Worker) leaveCluster(requestID string, timeout time.Duration) error {

	// Shutdown can take a while so use short locking intervals.
	cluster, ok := w.clusters.Get(requestID)
	if !ok {
		return errors.New("no cluster with that ID")
	}

	// TODO: Fix this logging.
	w.Log().Info().
		Stringer("consensus", cluster.Consensus()).
		Str("request", requestID).
		Msg("leaving consensus cluster")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// We know that the request is done executing when we have a result for it.
	_, ok = w.executeResponses.WaitFor(ctx, requestID)

	log := w.Log().With().Str("request", requestID).Logger()
	log.Info().Bool("executed_work", ok).Msg("waiting for execution done, leaving cluster")

	err := cluster.Shutdown()
	if err != nil {
		// Not much we can do at this point.
		return fmt.Errorf("could not leave cluster (request: %v): %w", requestID, err)
	}

	w.clusters.Delete(requestID)

	return nil
}

// helper function just for the sake of readibility.
func consensusRequired(c consensus.Type) bool {
	return c != 0
}