package apicontract

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// Replay is a closed transport: unexpected calls fail, never reach a network.
// Check also detects callers which stop before consuming every fixture page.
type Replay struct {
	Fixture  Fixture
	Examples []Example
	mu       sync.Mutex
	used     int
	failed   bool
}

func (r *Replay) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fail := func() (*http.Response, error) {
		r.failed = true
		return nil, errors.New("request does not match contract fixture")
	}
	if r.used >= len(r.Examples) {
		return fail()
	}
	ex := r.Examples[r.used]
	target, err := r.Fixture.URL("fixture-owner", "fixture-repo", ex.Query)
	if err != nil {
		return fail()
	}
	u, _ := url.Parse(target)
	if req.Method != r.Fixture.Method || req.URL.Scheme != u.Scheme || req.URL.Host != u.Host || req.URL.EscapedPath() != u.EscapedPath() || req.URL.Query().Encode() != u.Query().Encode() {
		return fail()
	}
	r.used++
	h := http.Header{}
	for k, v := range ex.Headers {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: ex.Status, Header: h, Body: io.NopCloser(bytes.NewReader(ex.Body)), Request: req}, nil
}

func (r *Replay) Check() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failed || r.used != len(r.Examples) {
		return errors.New("fixture exchange sequence was not fully matched")
	}
	return nil
}
