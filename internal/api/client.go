package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
)

const (
	BaseURL    = "https://api.atomgit.com"
	APIVersion = "/api/v5"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	ctx        context.Context
}

const defaultMetadataTimeout = 30 * time.Second

// HTTPError describes a non-successful API response. The Error method keeps
// the established "API error: <status> - <context>" message shape so existing
// callers and tests that match on the message keep working, while callers can
// still inspect the status code with IsHTTPStatus.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
	RetryAfter string
	ReadError  error
}

func (e *HTTPError) Error() string {
	message := fmt.Sprintf("API error: %s - %s", e.Status, e.Body)
	if e.ReadError != nil {
		message += " (failed to read error response: " + SanitizeErrorText(e.ReadError.Error()) + ")"
	}
	if e.RetryAfter != "" {
		message += " (retry after " + e.RetryAfter + ")"
	}
	return message
}

func (e *HTTPError) Unwrap() error {
	return e.ReadError
}

// IsHTTPStatus reports whether err is (or wraps) an API error carrying the
// given HTTP status code.
func IsHTTPStatus(err error, statusCode int) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == statusCode
}

func NewClient(token string) *Client {
	return NewClientWithHTTPClient(token, &http.Client{
		Timeout: defaultMetadataTimeout,
	})
}

// NewClientWithHTTPClient creates an API client using the provided HTTP client.
// A nil HTTP client uses the same default timeout as NewClient.
func NewClientWithHTTPClient(token string, httpClient *http.Client) *Client {
	return NewClientWithBaseURL(token, BaseURL+APIVersion, httpClient)
}

// NewClientWithBaseURL creates a client for a specific AtomGit API base URL.
// It is primarily used by clients for API versions other than v5 while keeping
// authentication headers, timeouts, and raw response handling consistent.
func NewClientWithBaseURL(token, baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultMetadataTimeout,
		}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: httpClient,
		ctx:        context.Background(),
	}
}

// WithContext returns a shallow copy of the client whose requests inherit ctx.
// The HTTP client and transport remain shared with the original client.
func (c *Client) WithContext(ctx context.Context) *Client {
	if c == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	clone := *c
	clone.ctx = ctx
	return &clone
}

func (c *Client) requestContext() context.Context {
	if c == nil || c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// metadataHTTPClient guarantees that regular JSON and raw metadata responses
// have a finite whole-request timeout, including response-body reads. Explicit
// shorter or longer non-zero caller timeouts are preserved.
func metadataHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return &http.Client{Timeout: defaultMetadataTimeout}
	}
	if client.Timeout > 0 {
		return client
	}
	clone := *client
	clone.Timeout = defaultMetadataTimeout
	return &clone
}

// streamingHTTPClient clones the configured client without its whole-request
// timeout. http.Client.Timeout includes reading the response body, so retaining
// the metadata client's 30-second limit would truncate large streamed bodies.
// Connection, TLS, and response-header timeouts remain enforced by the
// underlying transport.
func streamingHTTPClient(client *Client) *http.Client {
	base := client.httpClient
	if base == nil {
		base = &http.Client{}
	}

	streaming := *base
	streaming.Timeout = 0

	switch transport := streaming.Transport.(type) {
	case nil:
		if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
			cloned := defaultTransport.Clone()
			cloned.ResponseHeaderTimeout = 30 * time.Second
			streaming.Transport = cloned
		} else {
			streaming.Transport = http.DefaultTransport
		}
	case *http.Transport:
		cloned := transport.Clone()
		if cloned.ResponseHeaderTimeout == 0 {
			cloned.ResponseHeaderTimeout = 30 * time.Second
		}
		streaming.Transport = cloned
	}

	return &streaming
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithContentType(method, path, body, "application/json")
}

// isIdempotent reports whether the HTTP method is safe to retry on network errors.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodDelete, http.MethodPut:
		return true
	}
	return false
}

func (c *Client) doRequestWithContentType(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	return c.doRequestWithContentTypeAndAccept(method, path, body, contentType, "application/json")
}

func (c *Client) doRequestWithContentTypeAndAccept(method, path string, body io.Reader, contentType, accept string) (*http.Response, error) {
	return c.doRequestWithPolicy(c.httpClient, method, path, body, contentType, accept, isIdempotent(method))
}

// doRequestWithPolicy performs a request with an explicit HTTP client and
// retry policy. Most API calls use the client's normal idempotency policy;
// state-sensitive operations can opt out of an otherwise ambiguous retry.
func (c *Client) doRequestWithPolicy(
	httpClient *http.Client,
	method, path string,
	body io.Reader,
	contentType, accept string,
	canRetry bool,
) (*http.Response, error) {
	return c.doRequestWithPolicyContext(
		c.requestContext(),
		metadataHTTPClient(httpClient),
		method,
		path,
		body,
		contentType,
		accept,
		canRetry,
	)
}

// doRequestWithPolicyContext is doRequestWithPolicy with caller-controlled
// cancellation for streaming operations.
func (c *Client) doRequestWithPolicyContext(
	ctx context.Context,
	httpClient *http.Client,
	method, path string,
	body io.Reader,
	contentType, accept string,
	canRetry bool,
) (*http.Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("request context is nil")
	}

	// body 必须能被重读：io.Reader 读完一次就空了，重试时需重建 reader。
	// 把 body 读成 []byte 缓存，重试时用 bytes.NewReader 重建。GET/HEAD 无 body 不受影响。
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	requestURL := c.baseURL + path
	const retryDelay = 200 * time.Millisecond

	for attempt := 1; ; attempt++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
		if err != nil {
			return nil, err
		}

		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		req.Header.Set("User-Agent", "AtomCode-CLI/"+version.Get().Version)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if accept != "" {
			req.Header.Set("Accept", accept)
		}

		resp, err := httpClient.Do(req)
		// 首次失败 + 幂等方法 + 网络错误 → 短睡后重试一次
		if err != nil && canRetry && attempt == 1 && ctx.Err() == nil {
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}
		return resp, err
	}
}

func (c *Client) Get(path string, result interface{}) error {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return fmt.Errorf("API request GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return NewHTTPError(resp)
	}

	return decodeJSONResponse(http.MethodGet, path, resp.Body, result)
}

func (c *Client) Post(path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	resp, err := c.doRequest("POST", path, bodyReader)
	if err != nil {
		return fmt.Errorf("API request POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewHTTPError(resp)
	}

	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return decodeJSONResponse(http.MethodPost, path, resp.Body, result)
}

// PostForm sends an application/x-www-form-urlencoded POST request.
func (c *Client) PostForm(path string, fields url.Values, result interface{}) error {
	body := strings.NewReader(fields.Encode())
	resp, err := c.doRequestWithContentType(http.MethodPost, path, body, "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("API request POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewHTTPError(resp)
	}

	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return decodeJSONResponse(http.MethodPost, path, resp.Body, result)
}

// PutForm sends an application/x-www-form-urlencoded PUT request. Some
// endpoints (for example marking notifications read) only accept form-encoded
// bodies and reject the JSON encoding used by Put.
func (c *Client) PutForm(path string, fields url.Values, result interface{}) error {
	body := strings.NewReader(fields.Encode())
	resp, err := c.doRequestWithContentType(http.MethodPut, path, body, "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("API request PUT %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	if result == nil {
		return nil
	}
	return decodeJSONResponse(http.MethodPut, path, resp.Body, result)
}

func (c *Client) Put(path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	resp, err := c.doRequest("PUT", path, bodyReader)
	if err != nil {
		return fmt.Errorf("API request PUT %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	if result != nil {
		return decodeJSONResponse(http.MethodPut, path, resp.Body, result)
	}
	return nil
}

func (c *Client) Patch(path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	resp, err := c.doRequest("PATCH", path, bodyReader)
	if err != nil {
		return fmt.Errorf("API request PATCH %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewHTTPError(resp)
	}

	if result != nil {
		if err := decodeJSONResponse(http.MethodPatch, path, resp.Body, result); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}
	return nil
}

// PatchForm sends a multipart/form-data PATCH request.
func (c *Client) PatchForm(path string, fields map[string]string, result interface{}) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("encode form field %s: %w", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart form: %w", err)
	}

	resp, err := c.doRequestWithContentType(http.MethodPatch, path, &body, writer.FormDataContentType())
	if err != nil {
		return fmt.Errorf("API request PATCH %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	if result != nil {
		return decodeJSONResponse(http.MethodPatch, path, resp.Body, result)
	}
	return nil
}

func (c *Client) Delete(path string) error {
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("API request DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	return nil
}

func (c *Client) DeleteWithBody(path string, body interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	resp, err := c.doRequest("DELETE", path, bodyReader)
	if err != nil {
		return fmt.Errorf("API request DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	return nil
}

// DoRequestRaw performs a request and returns the raw *http.Response.
// The caller is responsible for closing resp.Body.
func (c *Client) DoRequestRaw(method, path string) (*http.Response, error) {
	return c.doRequest(method, path, nil)
}

// DoRequestRawWithAccept performs a request with a custom Accept header and
// returns the raw response. The caller is responsible for closing resp.Body.
func (c *Client) DoRequestRawWithAccept(method, path, accept string) (*http.Response, error) {
	return c.doRequestWithContentTypeAndAccept(method, path, nil, "", accept)
}

// DoRequestRawWithBody performs a request with replayable body bytes and
// caller-selected Content-Type and Accept headers. The caller is responsible
// for closing resp.Body.
func (c *Client) DoRequestRawWithBody(method, path string, body []byte, contentType, accept string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	return c.doRequestWithContentTypeAndAccept(method, path, bodyReader, contentType, accept)
}

// DoRequestRawStreamingWithAccept performs a cancellable raw request without
// the metadata client's whole-request timeout. It retains connection and
// response-header safeguards and is intended for large response bodies whose
// lifetime is controlled by the client's bound context.
func (c *Client) DoRequestRawStreamingWithAccept(method, path, accept string) (*http.Response, error) {
	return c.doRequestWithPolicyContext(
		c.requestContext(),
		streamingHTTPClient(c),
		method,
		path,
		nil,
		"",
		accept,
		isIdempotent(method),
	)
}

func decodeJSONResponse(method, path string, body io.Reader, result interface{}) error {
	if err := json.NewDecoder(body).Decode(result); err != nil {
		return fmt.Errorf("API request %s %s: decode response: %w", method, path, err)
	}
	return nil
}

// NewHTTPError reads a bounded excerpt of the response body and returns a
// terminal-safe, credential-redacted error preserving the established
// "API error: <status> - <context>" prefix. The returned error is an
// *HTTPError so callers can detect specific status codes with IsHTTPStatus.
func NewHTTPError(resp *http.Response) error {
	details := ReadErrorResponse(resp)
	return &HTTPError{
		StatusCode: resp.StatusCode,
		Status:     details.Status,
		Body:       details.Body,
		RetryAfter: details.RetryAfter,
		ReadError:  details.ReadError,
	}
}

// RequestPolicy configures how a single API request is dispatched.
// AllowedStatuses is the exact set of HTTP status codes treated as success;
// any other status produces a bounded, redacted API error. CanRetry controls
// whether the request is retried once on a network error.
type RequestPolicy struct {
	AllowedStatuses []int
	CanRetry        bool
}

func statusAllowed(code int, allowed []int) bool {
	for _, a := range allowed {
		if a == code {
			return true
		}
	}
	return false
}

// doJSONRequest performs an HTTP request with caller-selected retry policy
// and exact success statuses, decoding a JSON result when one is expected.
// Domain integrations use this primitive instead of the legacy helpers so
// they can request only their contracted 200 or 201 status and disable retry
// for state-sensitive operations such as related-branch PUT.
func (c *Client) doJSONRequest(method, path string, body io.Reader, contentType, accept string, policy RequestPolicy, result interface{}) error {
	return c.doJSONRequestContext(c.requestContext(), c.httpClient, method, path, body, contentType, accept, policy, result)
}

// doJSONRequestContext is doJSONRequest with caller-controlled cancellation
// and an explicit HTTP client.
func (c *Client) doJSONRequestContext(ctx context.Context, httpClient *http.Client, method, path string, body io.Reader, contentType, accept string, policy RequestPolicy, result interface{}) error {
	if len(policy.AllowedStatuses) == 0 {
		return fmt.Errorf("API request %s %s: allowed statuses cannot be empty", method, path)
	}

	resp, err := c.doRequestWithPolicyContext(ctx, metadataHTTPClient(httpClient), method, path, body, contentType, accept, policy.CanRetry)
	if err != nil {
		return fmt.Errorf("API request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if !statusAllowed(resp.StatusCode, policy.AllowedStatuses) {
		return NewHTTPError(resp)
	}

	if result == nil {
		return nil
	}
	return decodeJSONResponse(method, path, resp.Body, result)
}
