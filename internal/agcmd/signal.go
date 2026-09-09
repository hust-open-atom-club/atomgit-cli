package agcmd

import (
	"context"
	"os"
)

type notifyContextFunc func(context.Context, ...os.Signal) (context.Context, context.CancelFunc)

func newSignalContext(parent context.Context, notify notifyContextFunc) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return notify(parent, terminationSignals()...)
}
