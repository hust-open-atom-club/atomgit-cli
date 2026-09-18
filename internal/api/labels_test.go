package api

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestListLabelsEscapesOwnerAndRepo(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repos/org%2Fname/re%20po/labels" {
			t.Fatalf("path = %q", r.URL.EscapedPath())
		}
		_, _ = io.WriteString(w, `[{"name":"bug","color":"#ff0000"}]`)
	})

	labels, err := ListLabels(client, "org/name", "re po", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 1 || labels[0].Name != "bug" {
		t.Fatalf("labels = %#v", labels)
	}
}

func TestCreateLabelSendsFormAndRejectsRetry(t *testing.T) {
	requests := 0
	client := NewClient("token")
	client.baseURL = "https://example.test"
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/repos/alice/demo/labels" {
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
		}
		if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q", got)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if values.Get("name") != "bug" || values.Get("color") != "#ff0000" {
			t.Fatalf("form = %#v", values)
		}
		return nil, errors.New("connection reset")
	})}

	_, err := CreateLabel(client, "alice", "demo", CreateLabelRequest{Name: "bug", Color: "#ff0000"})
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestCreateLabelAcceptsCreatedJSON(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":1,"name":"bug","color":"#ff0000"}`)
	})

	label, err := CreateLabel(client, "alice", "demo", CreateLabelRequest{Name: "bug", Color: "#ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if label.Name != "bug" || label.Color != "#ff0000" {
		t.Fatalf("label = %#v", label)
	}
}

func TestUpdateLabelSendsMultipartWithoutRetry(t *testing.T) {
	requests := 0
	client := NewClient("token")
	client.baseURL = "https://example.test"
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/repos/alice/demo/labels/help%20wanted" {
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
		}
		if err := req.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if req.FormValue("name") != "support" || req.FormValue("color") != "#abc" {
			t.Fatalf("form = %#v", req.Form)
		}
		return nil, errors.New("connection reset")
	})}

	name := "support"
	color := "#abc"
	err := UpdateLabel(client, "alice", "demo", "help wanted", UpdateLabelRequest{Name: &name, Color: &color})
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestDeleteLabelAcceptsNoContentWithoutRetry(t *testing.T) {
	requests := 0
	client := NewClient("token")
	client.baseURL = "https://example.test"
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/repos/alice/demo/labels/obsolete" {
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
		}
		if requests == 1 {
			return nil, errors.New("connection reset")
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	err := DeleteLabel(client, "alice", "demo", "obsolete")
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestDeleteLabelAcceptsOK(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	})
	if err := DeleteLabel(client, "alice", "demo", "obsolete"); err != nil {
		t.Fatal(err)
	}
}

func TestListLabelsDoesNotAcceptCreated(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `[]`)
	})
	_, err := ListLabels(client, "alice", "demo", 10)
	if !IsHTTPStatus(err, http.StatusCreated) {
		t.Fatalf("error = %v, want HTTP 201", err)
	}
}
