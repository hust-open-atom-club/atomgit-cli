package api

import (
	"errors"
	"net/http"
	"testing"
)

func TestCustomRolesValidationErrorCauses(t *testing.T) {
	for _, tt := range []struct {
		name      string
		operation string
		causeText string
		request   func(*Client) error
	}{
		{
			name: "organization", operation: "organization customized roles",
			causeText: "roles[0].id must be a non-empty string",
			request: func(client *Client) error {
				_, err := GetOrganizationCustomRoles(client, "team")
				return err
			},
		},
		{
			name: "repository", operation: "repository customized roles",
			causeText: "roles[0].role_id must be a non-empty string",
			request: func(client *Client) error {
				_, err := GetRepositoryCustomRoles(client, "team", "demo")
				return err
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`[{}]`))
			})
			err := tt.request(client)
			if !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("missing invalid-response cause: %v", err)
			}
			want := ErrInvalidResponse.Error() + ": " + tt.operation + ": " + tt.causeText
			if err.Error() != want {
				t.Fatalf("diagnostic = %q, want %q", err, want)
			}

			var validationCause error
			for pending := []error{err}; len(pending) > 0; {
				current := pending[len(pending)-1]
				pending = pending[:len(pending)-1]
				if current == nil {
					continue
				}
				if current.Error() == tt.causeText {
					validationCause = current
					break
				}
				if joined, ok := current.(interface{ Unwrap() []error }); ok {
					pending = append(pending, joined.Unwrap()...)
				} else if cause := errors.Unwrap(current); cause != nil {
					pending = append(pending, cause)
				}
			}
			if validationCause == nil || !errors.Is(err, validationCause) {
				t.Fatalf("validation cause is not reachable: %v", err)
			}
		})
	}
}
