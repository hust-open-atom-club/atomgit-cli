package api

import (
	"encoding/json"
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
		{name: "nil", value: nil, want: "<nil>"},
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

func TestPullRequestReviewRequestJSON(t *testing.T) {
	tests := []struct {
		name  string
		event PullRequestReviewEvent
		body  string
		want  string
	}{
		{name: "approve without body", event: PullRequestReviewApprove, want: `{"event":"APPROVE"}`},
		{name: "request changes", event: PullRequestReviewRequestChanges, body: "Please add tests.", want: `{"body":"Please add tests.","event":"REQUEST_CHANGES"}`},
		{name: "comment", event: PullRequestReviewComment, body: "A few notes.", want: `{"body":"A few notes.","event":"COMMENT"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(PullRequestReviewRequest{Body: tt.body, Event: tt.event})
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}
