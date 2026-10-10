package root

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestDiscussionEditErrorsThroughRoot(t *testing.T) {
	f := &cmdutil.Factory{
		Config: rootTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: rootRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"d1","number":7,"title":"Old","md_content":"Body","category":{"name":"Ideas"}}`))}, nil
				}
				return &http.Response{StatusCode: 403, Status: "403 Forbidden", Body: io.NopCloser(strings.NewReader(`{"message":"denied\u001b[31m Bearer secret","access_token":"hidden-token"}`))}, nil
			})}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(f, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"discussion", "edit", "owner/repo", "7", "--title=Updated"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("failed edit succeeded")
	}
	fmt.Fprintln(cmd.ErrOrStderr(), err)
	if err := cmdutil.FlushWriter(cmd.ErrOrStderr()); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "failed to update discussion #7 for owner/repo") || !strings.Contains(stderr.String(), "Forbidden") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
	for _, unsafe := range []string{"secret", "hidden-token", "\x1b"} {
		if strings.Contains(stderr.String(), unsafe) {
			t.Fatalf("stderr contains %q: %q", unsafe, stderr.String())
		}
	}
}

func TestDiscussionEditHelpThroughRoot(t *testing.T) {
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"discussion", "edit", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"edit [<owner>/<repo>] <number>", "--title", "--body", "--body-file", "--category", "--body-file discussion.md", "--body-file -", "repository is inferred"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help missing %q", want)
		}
	}
}
