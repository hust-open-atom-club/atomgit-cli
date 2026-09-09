package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestListRepositoryForksPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/repos/team/demo/forks" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q", got)
		}
		if got := r.URL.Query().Get("page"); got != fmt.Sprint(requests) {
			t.Fatalf("page = %q, want %d", got, requests)
		}

		count, start := 100, 0
		if requests == 2 {
			count, start = 3, 100
		}
		forks := make([]repositoryForkResponse, count)
		for i := range forks {
			forks[i].FullName = fmt.Sprintf("alice/fork-%d", start+i)
		}
		if err := json.NewEncoder(w).Encode(forks); err != nil {
			t.Fatal(err)
		}
	})

	forks, err := ListRepositoryForks(client, "team", "demo", 103)
	if err != nil {
		t.Fatal(err)
	}
	if len(forks) != 103 || forks[102].Name != "fork-102" {
		t.Fatalf("forks = %d, last = %#v", len(forks), forks[len(forks)-1])
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestListRepositoryForksStopsAtShortPage(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewEncoder(w).Encode([]repositoryForkResponse{{FullName: "alice/fork-1"}}); err != nil {
			t.Fatal(err)
		}
	})

	forks, err := ListRepositoryForks(client, "team", "demo", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(forks) != 1 || requests != 1 {
		t.Fatalf("forks = %d, requests = %d", len(forks), requests)
	}
}

func TestListRepositoryForksMapsLiveResponseShape(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{
			"id":10658715,
			"full_name":"moyigeek/atomgit-cli",
			"url":"https://api.atomgit.com/api/v5/repos/moyigeek/atomgit-cli",
			"namespace":{"path":"moyigeek","html_url":"https://atomgit.com/moyigeek"},
			"owner":{"login":"moyigeek"},
			"parent":{"full_name":"hust-open-atom-club/atomgit-cli","url":"https://api.atomgit.com/api/v5/repos/hust-open-atom-club/atomgit-cli"},
			"private":false,
			"public":true
		}]`))
	})

	forks, err := ListRepositoryForks(client, "team", "demo", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(forks) != 1 {
		t.Fatalf("forks = %#v", forks)
	}
	fork := forks[0]
	if fork.Name != "atomgit-cli" || fork.FullName != "moyigeek/atomgit-cli" || fork.HTMLURL != "https://atomgit.com/moyigeek/atomgit-cli" {
		t.Fatalf("identity mapping = %#v", fork)
	}
	if !fork.Fork || fork.ParentFullName != "hust-open-atom-club/atomgit-cli" || fork.Owner.Login != "moyigeek" || fork.Namespace.Path != "moyigeek" {
		t.Fatalf("fork metadata mapping = %#v", fork)
	}
}

func TestListRepositoryForksEscapesPath(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/repos/owner%20with%20space/repo%2Fslash/forks" {
			t.Fatalf("path = %s", got)
		}
		_, _ = w.Write([]byte("[]"))
	})

	if _, err := ListRepositoryForks(client, "owner with space", "repo/slash", 1); err != nil {
		t.Fatal(err)
	}
}

func TestListRepositoryForksRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		if _, err := ListRepositoryForks(NewClient("token"), "team", "demo", limit); err == nil {
			t.Fatalf("limit %d was accepted", limit)
		}
	}
}

func TestListRepositoryForksPropagatesAPIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	})

	_, err := ListRepositoryForks(client, "team", "demo", 1)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}
