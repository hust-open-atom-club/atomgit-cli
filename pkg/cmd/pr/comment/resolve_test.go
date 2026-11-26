package comment

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type resolveCountingReader struct {
	io.Reader
	reads int
}

func (r *resolveCountingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func TestResolveAndUnresolveVerifyState(t *testing.T) {
	for _, resolved := range []bool{true, false} {
		for _, explicitRepo := range []bool{true, false} {
			for _, yes := range []bool{true, false} {
				t.Run(fmt.Sprintf("resolved=%t/explicit=%t/yes=%t", resolved, explicitRepo, yes), func(t *testing.T) {
					puts, reads := 0, 0
					f := &cmdutil.Factory{Config: prCommentTestConfig{userErr: errors.New("must not restrict to comment author")}, RepositoryResolver: func() (cmdutil.Repository, error) {
						if explicitRepo {
							t.Fatal("explicit repository must take precedence")
						}
						return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
					}, HTTPClient: func() (*http.Client, error) {
						return &http.Client{Transport: prCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
							if req.Method == http.MethodPut {
								puts++
								return prJSONResponse(200, ""), nil
							}
							if req.URL.Path == "/api/v5/repos/alice/demo/pulls/8" {
								return prJSONResponse(200, `{"number":8}`), nil
							}
							reads++
							value := !resolved
							if puts > 0 {
								value = resolved
							}
							return prJSONResponse(200, fmt.Sprintf(`[{"id":99,"discussion_id":"thread-id","comment_type":"diff_comment","resolved":%t}]`, value)), nil
						})}, nil
					}}
					cmd := newCmdResolve(f, resolved)
					args := []string{"8", "thread-id"}
					input := &resolveCountingReader{Reader: strings.NewReader("YES\n")}
					cmd.SetIn(input)
					if yes {
						args = append(args, "--yes")
					}
					if explicitRepo {
						args = append([]string{"alice/demo"}, args...)
					}
					var out, stderr bytes.Buffer
					cmd.SetOut(&out)
					cmd.SetErr(&stderr)
					cmd.SetArgs(args)
					if err := cmd.Execute(); err != nil {
						t.Fatal(err)
					}
					if puts != 1 || reads != 2 || !strings.Contains(out.String(), "(verified)") {
						t.Fatalf("puts=%d reads=%d out=%s", puts, reads, &out)
					}
					if yes {
						if input.reads != 0 || stderr.Len() != 0 {
							t.Fatalf("--yes read stdin %d times or prompted: %q", input.reads, stderr.String())
						}
					} else {
						want := fmt.Sprintf("Set resolved=%t for discussion thread-id on alice/demo PR #8? [y/N]: ", resolved)
						if input.reads == 0 || stderr.String() != want {
							t.Fatalf("stdin reads=%d prompt=%q want=%q", input.reads, stderr.String(), want)
						}
					}
				})
			}
		}
	}
}

func TestResolveInvalidAndCanceledBeforeAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		input string
		fail  bool
	}{
		{"missing", nil, "", true}, {"zero PR", []string{"alice/demo", "0", "thread"}, "yes\n", true},
		{"bad PR", []string{"alice/demo", "NaN", "thread"}, "yes\n", true},
		{"bad ID", []string{"alice/demo", "8", "../thread"}, "yes\n", true},
		{"blank ID", []string{"alice/demo", "8", " "}, "yes\n", true},
		{"bad repo", []string{"alice/demo/other", "8", "thread"}, "yes\n", true},
		{"cancel", []string{"alice/demo", "8", "thread"}, "n\n", false},
		{"EOF", []string{"alice/demo", "8", "thread"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &prCommentRecordingConfig{}
			cmd := newCmdResolve(&cmdutil.Factory{Config: cfg}, true)
			cmd.SilenceUsage, cmd.SilenceErrors = true, true
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetIn(strings.NewReader(tc.input))
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if (err != nil) != tc.fail || cfg.getTokenCalls != 0 {
				t.Fatalf("calls=%d err=%v", cfg.getTokenCalls, err)
			}
		})
	}
}

func TestResolveAuthenticationError(t *testing.T) {
	cmd := newCmdResolve(&cmdutil.Factory{Config: prCommentTestConfig{tokenErr: config.ErrNotAuthenticated}}, true)
	cmd.SetArgs([]string{"alice/demo", "8", "thread", "--yes"})
	if err := cmd.Execute(); !errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveFailuresNeverReportSuccess(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		getStatus, putStatus int
		before, after        string
		writes               int
	}{
		{"no permission", 403, 200, `[]`, `[]`, 0},
		{"PR missing", 404, 200, `[]`, `[]`, 0},
		{"note ID not discussion ID", 200, 200, `[{"id":42,"discussion_id":"other","comment_type":"diff_comment","resolved":false}]`, `[]`, 0},
		{"plain comment", 200, 200, `[{"discussion_id":"42","comment_type":"pr_comment","resolved":false}]`, `[]`, 0},
		{"write denied", 200, 403, `[{"discussion_id":"42","comment_type":"diff_comment","resolved":false}]`, `[]`, 1},
		{"readback unchanged", 200, 200, `[{"discussion_id":"42","comment_type":"diff_comment","resolved":false}]`, `[{"discussion_id":"42","comment_type":"diff_comment","resolved":false}]`, 1},
		{"readback missing state", 200, 200, `[{"discussion_id":"42","comment_type":"diff_comment","resolved":false}]`, `[{"discussion_id":"42","comment_type":"diff_comment"}]`, 1},
		{"readback wrong identity", 200, 200, `[{"discussion_id":"42","comment_type":"diff_comment","resolved":false}]`, `[]`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			puts := 0
			f := &cmdutil.Factory{Config: prCommentTestConfig{}, HTTPClient: func() (*http.Client, error) {
				return &http.Client{Transport: prCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodPut {
						puts++
						return prJSONResponse(tc.putStatus, ""), nil
					}
					if tc.getStatus != 200 {
						return prJSONResponse(tc.getStatus, `{}`), nil
					}
					if req.URL.Path == "/api/v5/repos/alice/demo/pulls/8" {
						return prJSONResponse(200, `{"number":8}`), nil
					}
					body := tc.before
					if puts > 0 {
						body = tc.after
					}
					if req.URL.Query().Get("page") != "1" {
						body = `[]`
					}
					return prJSONResponse(200, body), nil
				})}, nil
			}}
			cmd := newCmdResolve(f, true)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"alice/demo", "8", "42", "--yes"})
			if err := cmd.Execute(); err == nil || puts != tc.writes || out.Len() != 0 {
				t.Fatalf("puts=%d out=%s err=%v", puts, &out, err)
			}
		})
	}
}
