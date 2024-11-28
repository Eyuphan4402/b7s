package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blocklessnetwork/b7s/node"
	"github.com/blocklessnetwork/b7s/node/worker"
	"github.com/blocklessnetwork/b7s/testing/mocks"
)

const (
	loopback = "127.0.0.1"
)

func CreateWorkerNode(t *testing.T) *Worker {
	t.Helper()

	var (
		logger   = mocks.NoopLogger
		host     = NewLoopbackHost(t, logger)
		fstore   = mocks.BaselineFStore(t)
		executor = mocks.BaselineExecutor(t)

		core = node.NewCore(logger, host)
	)

	worker, err := worker.New(core, fstore, executor)
	require.NoError(t, err)

	return worker
}
