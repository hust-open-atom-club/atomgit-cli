package apicontract

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// Probe checks one page of a read-only endpoint. The transport argument allows
// deterministic offline testing; nil uses the standard HTTPS transport.
// Responses are discarded after shape validation, never recorded or printed.
func Probe(ctx context.Context, transport http.RoundTripper, f Fixture, owner, repo, token string) error {
	if f.Method != http.MethodGet {
		return errors.New("live contracts only allow GET")
	}
	if err := f.check(); err != nil {
		return err
	}
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return errors.New("dedicated contract token is required")
	}
	target, err := f.URL(owner, repo, f.Examples[0].Query)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return errors.New("cannot construct contract request")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(request)
	if err != nil {
		return errors.New("contract request failed (transport, TLS or timeout)")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		return errors.New("cannot read contract response")
	}
	return f.Validate(resp.StatusCode, resp.Header, body)
}
