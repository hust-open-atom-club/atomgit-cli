package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryTransferEndpointsAndBodies(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		body     map[string]string
		response string
		call     func(*Client) error
	}{
		{
			name:     "general repository",
			path:     "/repos/source/demo/transfer",
			body:     map[string]string{"new_owner": "destination"},
			response: `{"new_owner":"destination","new_name":"demo"}`,
			call: func(client *Client) error {
				response, err := TransferRepository(client, "source", "demo", "destination")
				if err == nil && (response.NewOwner != "destination" || response.NewName != "demo") {
					t.Fatalf("response = %#v", response)
				}
				return err
			},
		},
		{
			name:     "organization repository",
			path:     "/org/source-org/projects/demo/transfer",
			body:     map[string]string{"transfer_to": "destination", "password": "secret"},
			response: `{"code":1,"msg":"success"}`,
			call: func(client *Client) error {
				response, err := TransferOrganizationRepository(client, "source-org", "demo", "destination", "secret")
				if err == nil && (response.Code != 1 || response.Msg != "success") {
					t.Fatalf("response = %#v", response)
				}
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			client := NewClientWithHTTPClient("token", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5"+test.path {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				var body map[string]string
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body, test.body) {
					t.Fatalf("body = %#v, want %#v", body, test.body)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(test.response)),
					Request:    req,
				}, nil
			})})

			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
		})
	}
}
