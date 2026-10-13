package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"
)

// ErrInvalidUserIdentity indicates a successful response without a usable login.
// It deliberately excludes response data, which may contain credentials.
var ErrInvalidUserIdentity = errors.New("invalid user identity response")

// CurrentUserLogin verifies identity through the authenticated, read-only v5 API.
func (c *Client) CurrentUserLogin() (string, error) {
	// Only the authenticated user endpoint can establish identity. A redirect
	// may point to a public resource or forward credentials to another origin.
	// Clone the client so this stricter policy does not affect other commands.
	httpClient := *metadataHTTPClient(c.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := c.doRequestWithPolicy(&httpClient, http.MethodGet, "/user", nil, "", "application/json", true)
	if err != nil {
		return "", fmt.Errorf("verify current user: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Closing only releases response resources; status and response read errors are handled separately.
	if resp.StatusCode != http.StatusOK {
		// Status alone determines the verification failure. Reading an unused
		// error body could stall or replace a known 401/403/5xx with a timeout.
		// Exclude server-supplied status text and credential-bearing bodies.
		return "", &HTTPError{StatusCode: resp.StatusCode, Status: http.StatusText(resp.StatusCode)}
	}
	const maxSize = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return "", fmt.Errorf("read current user response: %w", err)
	}
	var user struct {
		Login string `json:"login"`
	}
	if len(data) > maxSize || json.Unmarshal(data, &user) != nil || user.Login == "" || strings.ContainsFunc(user.Login, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return "", ErrInvalidUserIdentity
	}
	return user.Login, nil
}
