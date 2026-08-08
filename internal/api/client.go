package api

import (
	"bytes"
	"context"
	"encoding/json"
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
}

func NewClient(token string) *Client {
	return NewClientWithHTTPClient(token, &http.Client{
		Timeout: 30 * time.Second,
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
			Timeout: 30 * time.Second,
		}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: httpClient,
	}
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
		context.Background(),
		httpClient,
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// PostForm sends an application/x-www-form-urlencoded POST request.
func (c *Client) PostForm(path string, fields url.Values, result interface{}) error {
	body := strings.NewReader(fields.Encode())
	resp, err := c.doRequestWithContentType(http.MethodPost, path, body, "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(responseBody))
	}

	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil && err != io.EOF {
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(responseBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) Delete(path string) error {
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
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
