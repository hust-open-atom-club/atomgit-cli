package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type recordingConfig struct {
	getTokenCalls int
}

func (r *recordingConfig) GetToken() (string, error) {
	r.getTokenCalls++
	return "token", nil
}
func (*recordingConfig) GetUser() (string, error) { return "alice", nil }
func (*recordingConfig) GetHost() string          { return "atomgit.com" }

type notificationAuthErrorConfig struct{}

func (notificationAuthErrorConfig) GetToken() (string, error) {
	return "", errors.New("not authenticated: run `ag auth login`")
}
func (notificationAuthErrorConfig) GetUser() (string, error) { return "alice", nil }
func (notificationAuthErrorConfig) GetHost() string          { return "atomgit.com" }

func TestNotificationCommandsPreserveCanonicalAuthenticationError(t *testing.T) {
	const want = "not authenticated: run `ag auth login`"

	list := newCmdNotificationList(&cmdutil.Factory{Config: notificationAuthErrorConfig{}})
	if err := list.RunE(list, []string{"owner/repo"}); err == nil || err.Error() != want {
		t.Fatalf("notification list error = %v, want %q", err, want)
	}

	markRead := newCmdNotificationMarkRead(&cmdutil.Factory{Config: notificationAuthErrorConfig{}})
	if err := markRead.RunE(markRead, []string{"owner/repo", "notification-id"}); err == nil || err.Error() != want {
		t.Fatalf("notification mark-read error = %v, want %q", err, want)
	}
}

// Issue #49 requires arguments/flags -> authentication -> execution, so
// invalid local input must be rejected before GetToken is ever called and an
// unauthenticated user sees a usage error instead of a login prompt.
func TestNotificationListRejectsUnresolvableRepositoryBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, nil)

	if err == nil || !strings.Contains(err.Error(), "unable to determine repository") {
		t.Fatalf("error = %v, want 'unable to determine repository'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; repository resolution must happen before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationListRejectsInvalidLimitBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	err := cmd.RunE(cmd, []string{"owner/repo"})

	if err == nil || !strings.Contains(err.Error(), "invalid limit") {
		t.Fatalf("error = %v, want 'invalid limit'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; an invalid limit must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationListRejectsInvalidSinceBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("since", "yesterday"); err != nil {
		t.Fatalf("set since: %v", err)
	}
	err := cmd.RunE(cmd, []string{"owner/repo"})

	if err == nil || !strings.Contains(err.Error(), "invalid --since timestamp") {
		t.Fatalf("error = %v, want 'invalid --since timestamp'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; an invalid --since must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationListRejectsInvalidBeforeBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("before", "soon"); err != nil {
		t.Fatalf("set before: %v", err)
	}
	err := cmd.RunE(cmd, []string{"owner/repo"})

	if err == nil || !strings.Contains(err.Error(), "invalid --before timestamp") {
		t.Fatalf("error = %v, want 'invalid --before timestamp'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; an invalid --before must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationMarkReadRejectsUnresolvableRepositoryBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationMarkRead(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"some-notification-id"})

	if err == nil || !strings.Contains(err.Error(), "unable to determine repository") {
		t.Fatalf("error = %v, want 'unable to determine repository'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; repository resolution must happen before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationMarkReadRejectsEmptySelectionBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationMarkRead(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"owner/repo"})

	if err == nil || !strings.Contains(err.Error(), "at least one notification ID or use --all") {
		t.Fatalf("error = %v, want empty selection error", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; an empty selection must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationMarkReadRejectsAllCombinedWithIDsBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationMarkRead(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set all: %v", err)
	}
	err := cmd.RunE(cmd, []string{"owner/repo", "some-notification-id"})

	if err == nil || !strings.Contains(err.Error(), "cannot combine --all with explicit notification IDs") {
		t.Fatalf("error = %v, want --all conflict error", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; the --all conflict must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestNotificationMarkReadRejectsBlankIDBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdNotificationMarkRead(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"owner/repo", " "})

	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("error = %v, want blank ID error", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; a blank ID must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestConfirmMarkAll(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "lowercase y", input: "y\n", want: true},
		{name: "uppercase yes", input: "YES\n", want: true},
		{name: "mixed yes", input: "Yes\n", want: true},
		{name: "no declines", input: "n\n", want: false},
		{name: "empty declines", input: "\n", want: false},
		{name: "eof declines", input: "", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := confirmMarkAll(strings.NewReader(tc.input), &out, 3, "owner/repo")
			if err != nil {
				t.Fatalf("confirmMarkAll: %v", err)
			}
			if got != tc.want {
				t.Fatalf("confirmed = %v, want %v", got, tc.want)
			}
			if !strings.Contains(out.String(), "Mark 3 unread notification(s) as read in owner/repo?") {
				t.Fatalf("prompt = %q", out.String())
			}
		})
	}
}

func TestPrintNotificationsEmptyState(t *testing.T) {
	var out bytes.Buffer
	printNotifications(&out, nil)
	if out.String() != "No notifications found\n" {
		t.Fatalf("output = %q, want empty state", out.String())
	}
}

func TestPrintNotificationsRows(t *testing.T) {
	notifications := []api.Notification{
		{
			ID:       "id-one",
			Content:  "fix: normalize unauthenticated error handling",
			Type:     "merge_requests_open",
			Unread:   true,
			UpdateAt: "2026-08-14T23:30:20+08:00",
			HTMLURL:  "https://gitcode.com/owner/repo/merge_requests/78",
		},
		{
			ID:       "id-two",
			Content:  "feat: support owner-scoped repository listing",
			Type:     "issue_open",
			Unread:   false,
			UpdateAt: "2026-08-13T10:00:00+08:00",
			HTMLURL:  "https://gitcode.com/owner/repo/issues/92",
		},
	}
	var out bytes.Buffer
	printNotifications(&out, notifications)

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if !strings.HasPrefix(lines[0], "unread\tmerge_requests_open\t2026-08-14T23:30:20+08:00\tfix: normalize unauthenticated error handling\t") {
		t.Fatalf("unread row = %q", lines[0])
	}
	if !strings.HasSuffix(lines[0], "https://gitcode.com/owner/repo/merge_requests/78") {
		t.Fatalf("unread row URL missing: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "read\tissue_open\t") {
		t.Fatalf("read row = %q", lines[1])
	}
}

func TestNotificationsJSON(t *testing.T) {
	notifications := []api.Notification{{
		ID:       "id-one",
		Content:  "subject text",
		Type:     "issue_open",
		Unread:   true,
		UpdateAt: "2026-08-14T23:30:20+08:00",
		HTMLURL:  "https://gitcode.com/owner/repo/issues/92",
	}}

	compact := func(t *testing.T, value string) string {
		t.Helper()
		var out bytes.Buffer
		if err := json.Compact(&out, []byte(value)); err != nil {
			t.Fatalf("compact %q: %v", value, err)
		}
		return out.String()
	}

	var buf bytes.Buffer
	if err := cmdutil.WriteJSON(&buf, notificationsJSON(notifications)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	want := `[{"id":"id-one","unread":true,"type":"issue_open","subject":"subject text","updatedAt":"2026-08-14T23:30:20+08:00","url":"https://gitcode.com/owner/repo/issues/92"}]`
	if got := compact(t, buf.String()); got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}

	// The JSON empty state must be a bare empty array.
	buf.Reset()
	if err := cmdutil.WriteJSON(&buf, notificationsJSON(nil)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if got := compact(t, buf.String()); got != "[]" {
		t.Fatalf("empty json = %s, want []", got)
	}
}
