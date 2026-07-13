package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
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
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Client{
		baseURL:    BaseURL + APIVersion,
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithContentType(method, path, body, "application/json")
}

func (c *Client) doRequestWithContentType(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "AtomCode-CLI-v0.4")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) Get(path string, result interface{}) error {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, cmdutil.SanitizeTerminal(string(body)))
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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, cmdutil.SanitizeTerminal(string(body)))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
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
		return fmt.Errorf("API error: %s - %s", resp.Status, cmdutil.SanitizeTerminal(string(body)))
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, cmdutil.SanitizeTerminal(string(body)))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
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
		return fmt.Errorf("API error: %s - %s", resp.Status, cmdutil.SanitizeTerminal(string(body)))
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
		return fmt.Errorf("API error: %s - %s", resp.Status, cmdutil.SanitizeTerminal(string(body)))
	}

	return nil
}

// DoRequestRaw performs a request and returns the raw *http.Response.
// The caller is responsible for closing resp.Body.
func (c *Client) DoRequestRaw(method, path string) (*http.Response, error) {
	return c.doRequest(method, path, nil)
}
