package release

import (
	"context"
	"fmt"
	"time"
)

const defaultReleaseTransferTimeout = 30 * time.Minute

func validateReleaseTransferTimeout(timeout time.Duration) error {
	if timeout < 0 {
		return fmt.Errorf("transfer timeout must not be negative")
	}
	return nil
}

func releaseTransferContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}
