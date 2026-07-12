package api

import "testing"

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
