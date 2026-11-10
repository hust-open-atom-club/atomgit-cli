package agcmd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type commandConfigStub struct{}

func (commandConfigStub) GetToken() (string, error) { return "stored-token", nil }
func (commandConfigStub) GetUser() (string, error)  { return "alice", nil }
func (commandConfigStub) GetHost() string           { return "atomgit.com" }

func TestNewSignalContextUsesPlatformSignalsWithoutRealDelivery(t *testing.T) {
	type signalContextTestKey struct{}
	parent := context.WithValue(context.Background(), signalContextTestKey{}, "parent")
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

func TestLoadCommandConfigAppliesTokenOverride(t *testing.T) {
	loadCalls := 0
	factory := &cmdutil.Factory{
		TokenOverride: "command-token",
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: statusTransport(func(req *http.Request) (*http.Response, error) {
				if got, want := req.Header.Get("Authorization"), "Bearer command-token"; got != want {
					t.Fatalf("Authorization = %q, want %q", got, want)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"login":"bob"}`)), Request: req}, nil
			})}, nil
		},
	}
	if err := loadCommandConfig(nil, factory, func() (config.Config, error) {
		loadCalls++
		return nil, errors.New("corrupt persistent credentials")
	}); err != nil {
		t.Fatal(err)
	}
	if loadCalls != 0 {
		t.Fatalf("persistent config loads = %d, want 0", loadCalls)
	}
	got, err := factory.Config.GetToken()
	if err != nil || got != "command-token" {
		t.Fatalf("GetToken() = %q, %v", got, err)
	}
	user, err := factory.Config.GetUser()
	if err != nil || user != "bob" {
		t.Fatalf("GetUser() = %q, %v; want command-token identity", user, err)
	}
	if host := factory.Config.GetHost(); host != "atomgit.com" {
		t.Fatalf("GetHost() = %q, want default host", host)
	}
}

func TestLoadCommandConfigLoadsStoredConfigWithoutOverride(t *testing.T) {
	loadCalls := 0
	factory := &cmdutil.Factory{}
	if err := loadCommandConfig(nil, factory, func() (config.Config, error) {
		loadCalls++
		return commandConfigStub{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if loadCalls != 1 {
		t.Fatalf("persistent config loads = %d, want 1", loadCalls)
	}
	if token, err := factory.Config.GetToken(); err != nil || token != "stored-token" {
		t.Fatalf("GetToken() = %q, %v", token, err)
	}
	if user, err := factory.Config.GetUser(); err != nil || user != "alice" {
		t.Fatalf("GetUser() = %q, %v", user, err)
	}
}
