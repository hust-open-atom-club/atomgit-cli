package agcmd

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSignalContextUsesPlatformSignalsWithoutRealDelivery(t *testing.T) {
	parent := context.WithValue(context.Background(), struct{}{}, "parent")
	wantSignals := terminationSignals()
	var gotParent context.Context
	var gotSignals []os.Signal

	ctx, cancel := newSignalContext(parent, func(ctx context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		gotParent = ctx
		gotSignals = append([]os.Signal(nil), signals...)
		return context.WithCancel(ctx)
	})

	if gotParent != parent {
		t.Fatal("signal notifier did not receive the parent context")
	}
	if !reflect.DeepEqual(gotSignals, wantSignals) {
		t.Fatalf("signals = %v, want %v", gotSignals, wantSignals)
	}
	if len(gotSignals) == 0 || gotSignals[0] != os.Interrupt {
		t.Fatalf("signals = %v, want os.Interrupt first", gotSignals)
	}

	cancel()
	if err := ctx.Err(); err != context.Canceled {
		t.Fatalf("context error = %v, want context canceled", err)
	}
}

func TestIsExtensionCommand(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	extension := &cobra.Command{Use: "extension", GroupID: "extension"}
	regular := &cobra.Command{Use: "regular"}
	root.AddCommand(extension, regular)

	if !isExtensionCommand(root, []string{"extension"}) {
		t.Fatal("extension command was not recognized")
	}
	if isExtensionCommand(root, []string{"regular"}) {
		t.Fatal("regular command was recognized as an extension")
	}
	if isExtensionCommand(root, []string{"missing"}) {
		t.Fatal("missing command was recognized as an extension")
	}
}
