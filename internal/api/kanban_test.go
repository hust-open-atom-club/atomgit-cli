package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type kanbanRoundTripFunc func(*http.Request) (*http.Response, error)

func (f kanbanRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func kanbanResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func kanbanClient(transport kanbanRoundTripFunc) *Client {
	return NewClientWithHTTPClient("token", &http.Client{Transport: transport})
}

func TestValidateKanbanInputs(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
		want  string
		err   string
	}{
		{name: "owner", value: " team ", want: "team"},
		{name: "owner slash", value: "team/repo", err: "invalid organization owner"},
		{name: "owner empty", value: " ", err: "organization owner is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateKanbanOwner(tt.value)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error = %v, want %q", err, tt.err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, err %v; want %q", got, err, tt.want)
			}
		})
	}
	for _, value := range []string{"", "0", "-1", "abc", "1/2"} {
		if _, err := ValidateKanbanID(value); err == nil {
			t.Fatalf("ValidateKanbanID(%q) succeeded", value)
		}
	}
	if got, err := ValidateKanbanID(" 123 "); err != nil || got != "123" {
		t.Fatalf("ValidateKanbanID = %q, %v", got, err)
	}
}

func TestGetKanbansPaginatesEnvelopeAndHonorsLimit(t *testing.T) {
	requests := 0
	client := kanbanClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/api/v5/org/team/kanban/list" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		wantPerPage := "100"
		if requests == 2 {
			wantPerPage = "1"
		}
		if req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != wantPerPage {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		if requests == 1 {
			content := make([]Kanban, 100)
			for i := range content {
				iid := i + 1
				content[i] = Kanban{ID: fmt.Sprint(i + 1), IID: &iid, Name: fmt.Sprintf("board-%d", i+1)}
			}
			body, err := json.Marshal(kanbanListResponse{AllCount: 101, Content: content})
			if err != nil {
				t.Fatal(err)
			}
			return kanbanResponse(http.StatusOK, string(body)), nil
		}
		return kanbanResponse(http.StatusOK, `{"all_count":101,"content":[{"id":"3","iid":3,"name":"three"}]}`), nil
	})
	boards, err := GetKanbans(client, "team", 101)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(boards) != 101 || boards[100].Name != "three" {
		t.Fatalf("requests=%d boards=%#v", requests, boards)
	}
}

func TestKanbanItemColumnIgnoresSourceItemStatus(t *testing.T) {
	var item KanbanItem
	if err := json.Unmarshal([]byte(`{"id":1,"number":42,"title":"Issue","source_type":"issue","status":"closed","values":[{"field_name":"状态","field_type":"status"},{"field_name":"源数据状态","field_type":"artsStatus","value":"closed"}]}`), &item); err != nil {
		t.Fatal(err)
	}
	if got := item.Column(); got != "" {
		t.Fatalf("Column() = %q, want empty for source item status", got)
	}
}

func TestGetKanbanDetailAndItems(t *testing.T) {
	requests := 0
	client := kanbanClient(func(req *http.Request) (*http.Response, error) {
		requests++
		switch req.URL.Path {
		case "/api/v5/org/team/kanban/123/detail":
			return kanbanResponse(http.StatusOK, `{"id":"123","iid":7,"name":"Board","status":0}`), nil
		case "/api/v5/org/team/kanban/123/item_list":
			if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "10" {
				t.Fatalf("query = %q", req.URL.RawQuery)
			}
			return kanbanResponse(http.StatusOK, `[{"id":1,"number":42,"title":"Issue","source_type":"issue","status":"opened","values":[{"field_name":"状态","field_type":"status","value":"Doing"}]},{"id":2,"number":"9","title":"PR","source_type":"pull_request","status":"closed"}]`), nil
		default:
			t.Fatalf("unexpected path %q", req.URL.Path)
			return nil, nil
		}
	})
	board, err := GetKanban(client, "team", "123")
	if err != nil || board.Name != "Board" {
		t.Fatalf("board=%#v err=%v", board, err)
	}
	items, err := GetKanbanItems(client, "team", "123", 10)
	if err != nil || len(items) != 2 || items[0].GetNumber() != "42" || items[0].Column() != "Doing" {
		t.Fatalf("items=%#v err=%v", items, err)
	}
	if requests != 2 {
		t.Fatalf("requests=%d, want 2", requests)
	}
}

func TestGetKanbanReportsAPIErrorAndMalformedResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
		want string
	}{
		{name: "permission", body: `{"message":"denied"}`, want: "403"},
		{name: "malformed", body: `{`, want: "unexpected EOF"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := kanbanClient(func(*http.Request) (*http.Response, error) {
				status := http.StatusForbidden
				if tt.name == "malformed" {
					status = http.StatusOK
				}
				return kanbanResponse(status, tt.body), nil
			})
			_, err := GetKanban(client, "team", "123")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
