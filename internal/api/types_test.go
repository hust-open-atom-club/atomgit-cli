package api

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNumberFormatting(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "string", value: "12", want: "12"},
		{name: "float64", value: float64(12), want: "12"},
		{name: "int", value: 12, want: "12"},
		{name: "int64", value: int64(12), want: "12"},
		{name: "other", value: true, want: "true"},
		{name: "nil", value: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&PullRequest{Number: tt.value}).GetNumber(); got != tt.want {
				t.Fatalf("PullRequest.GetNumber() = %q, want %q", got, tt.want)
			}
			if got := (&Issue{Number: tt.value}).GetNumber(); got != tt.want {
				t.Fatalf("Issue.GetNumber() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteResponseFixtures(t *testing.T) {
	t.Run("pull request create", func(t *testing.T) {
		data, err := os.ReadFile("testdata/pull_request_create_response.json")
		if err != nil {
			t.Fatal(err)
		}
		var result PullRequestWriteResponse
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if got := result.GetNumber(); got != "7" {
			t.Fatalf("number = %q, want 7", got)
		}
		if got := result.GetURL(); got != "https://atomgit.com/alice/demo/pull/7" {
			t.Fatalf("URL = %q", got)
		}
	})

	for _, tt := range []struct {
		name string
		file string
		want string
	}{
		{name: "issue comment create", file: "testdata/issue_comment_create_response.json", want: "180041703"},
		{name: "PR comment create", file: "testdata/pull_request_comment_create_response.json", want: "180041704"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatal(err)
			}
			var result CreateCommentResponse
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			if got := result.GetID(); got != tt.want {
				t.Fatalf("ID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPullRequestReviewRequestJSON(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		want  string
	}{
		{name: "normal approval", want: `{"force":false}`},
		{name: "forced approval", force: true, want: `{"force":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(PullRequestReviewRequest{Force: tt.force})
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}
