package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type identityReadFailure struct{ Err error }

func (r identityReadFailure) Read([]byte) (int, error) { return 0, r.Err }
func (identityReadFailure) Close() error               { return nil }
func TestIdentityDecoderPreservesCause(t *testing.T) {
	for _, body := range []string{`{"login":123}`, `{"login":`} {
		t.Run(body, func(t *testing.T) {
			client := NewClientWithHTTPClient("fixture",
				&http.Client{Transport: identityTransport(func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)),
						Header: make(http.Header), Request: req}, nil
				})})
			_, err := client.CurrentUserLogin()
			_, syntax := errors.AsType[*json.SyntaxError](err)
			_, wrongType := errors.AsType[*json.UnmarshalTypeError](err)
			if !errors.Is(err, ErrInvalidUserIdentity) || (!syntax && !wrongType) {
				t.Fatalf("lost identity or decoder cause: %v", err)
			}
		})
	}
}
func TestIdentityRequestStagesPreserveCause(t *testing.T) {
	cause := errors.New("private diagnostic")
	for _, stage := range []string{"prepare", "execute", "read"} {
		t.Run(stage, func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("fixture",
				&http.Client{Transport: identityTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					if stage == "execute" {
						return nil, cause
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: identityReadFailure{cause},
						Request: req}, nil
				})})
			if stage == "prepare" {
				client.baseURL = "://invalid"
			}
			_, err := client.CurrentUserLogin()
			switch stage {
			case "prepare":
				if !errors.Is(err, ErrRequestPreparation) || calls != 0 {
					t.Fatalf("wrong preparation contract: %v", err)
				}
				if _, ok := errors.AsType[*url.Error](err); !ok {
					t.Fatal("native parse cause lost")
				}
			case "execute":
				if _, ok := errors.AsType[*RequestExecutionError](err); !ok || !errors.Is(err, cause) {
					t.Fatalf("wrong execution contract: %v", err)
				}
			case "read":
				if _, ok := errors.AsType[*ResponseReadError](err); !ok || !errors.Is(err, cause) {
					t.Fatalf("wrong read contract: %v", err)
				}
			}
		})
	}
}
