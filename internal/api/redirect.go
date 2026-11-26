package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type anonymousRedirectKey struct{}

// RedirectSafeHTTPClient copies source and confines redirected credentials to
// their original origin. Caller redirect refusals and client settings survive.
// A transport guard runs after CookieJar processing, which follows CheckRedirect.
func RedirectSafeHTTPClient(source *http.Client) *http.Client {
	if source == nil {
		source = http.DefaultClient
	}
	clone := *source
	previous := source.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		stripRedirectCredentials(req, via)
		if previous != nil {
			if err := previous(req, via); err != nil {
				return err
			}
		}
		// A callback may change the destination or add headers. Enforce again.
		stripRedirectCredentials(req, via)
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if req.URL == nil {
			return fmt.Errorf("redirect destination is missing")
		}
		if strings.EqualFold(req.URL.Scheme, "http") {
			for _, prior := range via {
				if prior.URL != nil && strings.EqualFold(prior.URL.Scheme, "https") {
					return fmt.Errorf("HTTPS to HTTP redirect is not allowed")
				}
			}
			if !redirectLoopback(req.URL.Hostname()) {
				return fmt.Errorf("HTTP redirects are only allowed to loopback hosts")
			}
		}
		return nil
	}
	transport := source.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = redirectCredentialTransport{base: transport}
	return &clone
}

func stripRedirectCredentials(req *http.Request, via []*http.Request) {
	strip, _ := req.Context().Value(anonymousRedirectKey{}).(bool)
	for _, prior := range via {
		anonymous, _ := prior.Context().Value(anonymousRedirectKey{}).(bool)
		if anonymous || !redirectSameOrigin(prior.URL, req.URL) ||
			(len(via) > 0 && via[0].Header.Get("Authorization") != "" && prior.Header.Get("Authorization") == "") {
			strip = true
		}
	}
	if strip {
		removeRedirectCredentials(req.Header)
		*req = *req.WithContext(context.WithValue(req.Context(), anonymousRedirectKey{}, true))
	}
}

func removeRedirectCredentials(header http.Header) {
	for name := range header {
		switch strings.ToLower(name) {
		case "authorization", "proxy-authorization", "cookie", "cookie2":
			delete(header, name)
		}
	}
}

type redirectCredentialTransport struct{ base http.RoundTripper }

func (t redirectCredentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if anonymous, _ := req.Context().Value(anonymousRedirectKey{}).(bool); anonymous {
		clone := req.Clone(req.Context())
		removeRedirectCredentials(clone.Header)
		// URL user info would otherwise be converted into Basic Authorization.
		clone.URL.User = nil
		return t.base.RoundTrip(clone)
	}
	return t.base.RoundTrip(req)
}

func redirectSameOrigin(a, b *url.URL) bool {
	return a != nil && b != nil && strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Hostname(), b.Hostname()) && redirectPort(a) == redirectPort(b)
}

func redirectPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		if number, err := strconv.Atoi(port); err == nil {
			return strconv.Itoa(number)
		}
		return port
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(u.Scheme, "http") {
		return "80"
	}
	return ""
}

func redirectLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}
