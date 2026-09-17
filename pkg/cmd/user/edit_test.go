package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func userEditTestFactory(t *testing.T, handler http.HandlerFunc) *cmdutil.Factory {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport := server.Client().Transport
	return userFactory(userTestConfig{}, userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = target.Scheme
		cloned.URL.Host = target.Host
		cloned.Host = target.Host
		return serverTransport.RoundTrip(cloned)
	}))
}

func TestUserEditSendsEachExplicitFlag(t *testing.T) {
	tests := []struct {
		flag string
		key  string
	}{
		{flag: "avatar", key: "avatar"},
		{flag: "nickname", key: "nickname"},
		{flag: "company", key: "company"},
		{flag: "description", key: "description"},
		{flag: "email", key: "email"},
		{flag: "github-account", key: "github_account"},
		{flag: "website", key: "website"},
		{flag: "location", key: "location"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			factory := userEditTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPatch || req.URL.Path != "/api/v5/user" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("authorization = %q", got)
				}
				var body map[string]string
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if want := map[string]string{tt.key: "value"}; !reflect.DeepEqual(body, want) {
					t.Fatalf("body = %#v, want %#v", body, want)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"1","login":"alice"}`)
			})

			cmd := newCmdUserEdit(factory)
			cmd.SetArgs([]string{"--" + tt.flag, "value"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUserEditPartialUpdateAndExplicitClearing(t *testing.T) {
	factory := userEditTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"nickname": "Alice", "description": ""}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"1","login":"alice","nickname":"Alice","description":""}`)
	})
	cmd := newCmdUserEdit(factory)
	cmd.SetArgs([]string{"--nickname", "Alice", "--description", ""})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestUserEditNoFlagsFailsBeforeConfigurationOrNetwork(t *testing.T) {
	configErr := errors.New("configuration should not be read")
	transportCalled := false
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		transportCalled = true
		return nil, errors.New("network should not be used")
	})
	cmd := newCmdUserEdit(userFactory(userTestConfig{tokenErr: configErr}, transport))
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one of") {
		t.Fatalf("error = %v", err)
	}
	if errors.Is(err, configErr) {
		t.Fatalf("configuration was read: %v", err)
	}
	if transportCalled {
		t.Fatal("network was used")
	}
}

func TestUserEditWrapsAPIError(t *testing.T) {
	requestErr := errors.New("connection reset")
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, requestErr
	})
	cmd := newCmdUserEdit(userFactory(userTestConfig{}, transport))
	cmd.SetArgs([]string{"--nickname", "Alice"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to update user profile") {
		t.Fatalf("error = %v", err)
	}
	if !errors.Is(err, requestErr) {
		t.Fatalf("error did not retain cause: %v", err)
	}
}

func TestUserEditTextOutput(t *testing.T) {
	factory := userEditTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"1","login":"alice","nickname":"Alice"}`)
	})
	cmd := newCmdUserEdit(factory)
	cmd.SetArgs([]string{"--nickname", "Alice"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "✓ Updated user profile for alice\n  URL: https://atomgit.com/alice\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestUserEditJSONOutput(t *testing.T) {
	response := `{"avatar":"https://example.com/a.png","nickname":"Alice","company":"Example","description":"Hello","email":"alice@example.com","github_account":"alice-gh","website":"https://example.com","location":"Wuhan","id":"1","login":"alice"}`
	factory := userEditTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	})
	cmd := newCmdUserEdit(factory)
	cmd.SetArgs([]string{"--nickname", "Alice", "--json"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var got map[string]string
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"avatar": "https://example.com/a.png", "nickname": "Alice", "company": "Example",
		"description": "Hello", "email": "alice@example.com", "github_account": "alice-gh",
		"website": "https://example.com", "location": "Wuhan", "id": "1", "login": "alice",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output = %#v, want %#v", got, want)
	}
}

func TestUserEditRejectsArguments(t *testing.T) {
	cmd := newCmdUserEdit(userFactory(userTestConfig{}, nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"unexpected", "--nickname", "Alice"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("user edit accepted an argument")
	}
}

func TestNewCmdUserRegistersEdit(t *testing.T) {
	cmd := NewCmdUser(userFactory(userTestConfig{}, nil))
	edit, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"avatar", "nickname", "company", "description", "email", "github-account", "website", "location", "json"} {
		if edit.Flags().Lookup(flag) == nil {
			t.Fatalf("edit flag %q was not registered", flag)
		}
	}
}
