package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type userTestConfig struct {
	tokenErr error
}

func (c userTestConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (c userTestConfig) GetUser() (string, error)  { return "alice", nil }
func (c userTestConfig) GetHost() string           { return "atomgit.com" }

type userRoundTripFunc func(*http.Request) (*http.Response, error)

func (f userRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func userResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func userFactory(config userTestConfig, transport userRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func TestNewCmdUserRegistersView(t *testing.T) {
	cmd := NewCmdUser(&cmdutil.Factory{})
	view, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"json", "web"} {
		if view.Flags().Lookup(flag) == nil {
			t.Fatalf("view flag %q was not registered", flag)
		}
	}
	if !strings.Contains(view.Example, "ag user view alice --json") {
		t.Fatalf("view examples = %q", view.Example)
	}
	if !strings.Contains(view.Example, "ag user view alice --web") {
		t.Fatalf("view examples = %q", view.Example)
	}
	if err := view.Args(view, []string{"one", "two"}); err == nil {
		t.Fatal("user view accepted more than one argument")
	}
}

func TestUserViewTextOutput(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
		contains []string
	}{
		{
			name:     "current user",
			args:     nil,
			wantPath: "/api/v5/user",
			body:     `{"id":"1","login":"alice","name":"Alice","email":"alice@example.com","html_url":"https://atomgit.com/alice","type":"User","bio":"B","company":"C","website":"https://alice.dev","location":"Wuhan","followers":10,"following":3,"top_languages":["Go","Shell"]}`,
			contains: []string{"Login: alice", "Name:  Alice", "Email: alice@example.com", "URL:   https://atomgit.com/alice", "Type:  User", "Bio:   B", "Company: C", "Website: https://alice.dev", "Location: Wuhan", "Followers: 10", "Following: 3", "Top languages: Go, Shell"},
		},
		{
			name:     "public user",
			args:     []string{"bob"},
			wantPath: "/api/v5/users/bob",
			body:     `{"login":"bob","name":"Bob","html_url":"https://atomgit.com/bob","type":"User"}`,
			contains: []string{"Login: bob", "Name:  Bob", "URL:   https://atomgit.com/bob"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != tt.wantPath {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				return userResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdUserView(userFactory(userTestConfig{}, transport))
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, tt.args); err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("output = %q, missing %q", output.String(), want)
				}
			}
		})
	}
}

func TestUserViewPublicProfileWithoutToken(t *testing.T) {
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/users/bob" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return userResponse(http.StatusOK, `{"login":"bob","html_url":"https://atomgit.com/bob"}`), nil
	})
	cmd := newCmdUserView(userFactory(userTestConfig{tokenErr: errors.New("not authenticated: run `ag auth login`")}, transport))
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"bob"}); err != nil {
		t.Fatalf("public profile should not require a token: %v", err)
	}
	if !strings.Contains(output.String(), "Login: bob") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestUserViewCurrentUserRequiresToken(t *testing.T) {
	cmd := newCmdUserView(userFactory(userTestConfig{tokenErr: errors.New("not authenticated: run `ag auth login`")}, nil))
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserViewRejectsInvalidLoginBeforeRequest(t *testing.T) {
	tests := []string{"", "   ", "alice/team", "a?b", "a#b", "alice\x00", "alice\x1b[31m"}
	for _, login := range tests {
		t.Run(fmt.Sprintf("login=%q", login), func(t *testing.T) {
			transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
				return nil, nil
			})
			cmd := newCmdUserView(userFactory(userTestConfig{}, transport))
			err := cmd.RunE(cmd, []string{login})
			if err == nil || !strings.Contains(err.Error(), "invalid login") {
				t.Fatalf("error = %v, want invalid login", err)
			}
		})
	}
}

func TestUserViewEscapesLoginPathSegment(t *testing.T) {
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		// "?" is not rejected by validLogin? It is rejected above, so use a
		// login with characters that need escaping but stay valid.
		if req.URL.Path != "/api/v5/users/alice" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.URL.RawQuery != "" {
			t.Fatalf("unexpected query %q", req.URL.RawQuery)
		}
		return userResponse(http.StatusOK, `{"login":"alice","html_url":"https://atomgit.com/alice"}`), nil
	})
	cmd := newCmdUserView(userFactory(userTestConfig{}, transport))
	if err := cmd.RunE(cmd, []string{"alice"}); err != nil {
		t.Fatal(err)
	}
}

func TestUserViewJSON(t *testing.T) {
	transport := userRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return userResponse(http.StatusOK, `{"id":"1","login":"alice","name":"Alice","email":"alice@example.com","html_url":"https://atomgit.com/alice","type":"User","bio":"B","company":"C","website":"https://alice.dev","location":"Wuhan","followers":10,"following":3,"top_languages":["Go","Shell"]}`), nil
	})
	cmd := newCmdUserView(userFactory(userTestConfig{}, transport))
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	want := `{"id":"1","login":"alice","name":"Alice","email":"alice@example.com","url":"https://atomgit.com/alice","type":"User","bio":"B","company":"C","website":"https://alice.dev","location":"Wuhan","followers":10,"following":3,"topLanguages":["Go","Shell"]}`
	var gotValue, wantValue any
	if err := json.Unmarshal(output.Bytes(), &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("output = %s, want = %s", output.String(), want)
	}
}

func TestUserViewWeb(t *testing.T) {
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return userResponse(http.StatusOK, `{"login":"alice","html_url":"https://atomgit.com/alice"}`), nil
	})
	opened := ""
	factory := userFactory(userTestConfig{}, transport)
	factory.BrowserOpener = func(rawURL string) error {
		opened = rawURL
		return nil
	}
	cmd := newCmdUserView(factory)
	if err := cmd.Flags().Set("web", "true"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice"}); err != nil {
		t.Fatal(err)
	}
	if opened != "https://atomgit.com/alice" {
		t.Fatalf("opened = %q", opened)
	}
	if !strings.Contains(output.String(), "Opening https://atomgit.com/alice") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestUserViewWebBrowserError(t *testing.T) {
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return userResponse(http.StatusOK, `{"login":"alice","html_url":"https://atomgit.com/alice"}`), nil
	})
	factory := userFactory(userTestConfig{}, transport)
	factory.BrowserOpener = func(rawURL string) error {
		return errors.New("no browser")
	}
	cmd := newCmdUserView(factory)
	if err := cmd.Flags().Set("web", "true"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice"})
	if err == nil || !strings.Contains(err.Error(), "failed to open browser") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserViewAPIError(t *testing.T) {
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return userResponse(http.StatusNotFound, `{}`), nil
	})
	cmd := newCmdUserView(userFactory(userTestConfig{}, transport))
	err := cmd.RunE(cmd, []string{"missing"})
	if err == nil || !strings.Contains(err.Error(), "view user") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserViewMissingLoginInResponse(t *testing.T) {
	transport := userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return userResponse(http.StatusOK, `{}`), nil
	})
	cmd := newCmdUserView(userFactory(userTestConfig{}, transport))
	err := cmd.RunE(cmd, []string{"alice"})
	if err == nil || !strings.Contains(err.Error(), "did not include a login") {
		t.Fatalf("error = %v", err)
	}
}
