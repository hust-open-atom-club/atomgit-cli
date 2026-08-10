package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	baseapi "atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

const (
	APIVersion   = "/api/v8"
	maxErrorBody = 64 << 10
)

type Client struct {
	client *baseapi.Client
}

type ListRunsOptions struct {
	Event         string
	Status        string
	Branch        string
	Executor      string
	PullRequestID string
	WorkflowID    string
	WorkflowName  string
	Page          int
	PerPage       int
	StartTime     int64
	EndTime       int64
}

type ListArtifactsOptions struct {
	Name      string
	Sort      string
	Direction string
	Page      int
	PerPage   int
}

type ListWorkflowsOptions struct {
	Page    int
	PerPage int
}

type HTTPError struct {
	Operation  string
	StatusCode int
	Status     string
	Message    string
	RetryAfter string
	ReadError  error
}

func (e *HTTPError) Error() string {
	description := "request failed"
	switch e.StatusCode {
	case http.StatusForbidden:
		description = "permission denied"
	case http.StatusNotFound:
		description = "not found"
	case http.StatusTooManyRequests:
		description = "rate limited"
	}

	message := fmt.Sprintf("%s: %s (%s)", e.Operation, description, e.Status)
	if e.Message != "" {
		message += ": " + e.Message
	}
	if e.ReadError != nil {
		readMessage := "failed to read error response: " + e.ReadError.Error()
		if e.Message == "" {
			message += ": " + readMessage
		} else {
			message += " (" + readMessage + ")"
		}
	}
	if e.RetryAfter != "" {
		message += " (retry after " + e.RetryAfter + ")"
	}
	return message
}

func (e *HTTPError) Unwrap() error {
	return e.ReadError
}

func NewClient(token string) *Client {
	return NewClientWithHTTPClient(token, nil)
}

func NewClientWithHTTPClient(token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}
	return newClientWithBaseURL(token, baseapi.BaseURL+APIVersion, httpClient)
}

func defaultHTTPClient() *http.Client {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Transport: http.DefaultTransport}
	}
	transport := baseTransport.Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	// Do not set http.Client.Timeout: it includes reading the response body and
	// would truncate large streamed logs or artifacts. Connection, TLS, and
	// response-header timeouts remain enforced by the transport.
	return &http.Client{Transport: transport}
}

func newClientWithBaseURL(token, baseURL string, httpClient *http.Client) *Client {
	// Keep credentials in the shared Authorization header instead of the
	// documented access_token query parameter so URLs, redirects, and errors do
	// not expose the token.
	return &Client{
		client: baseapi.NewClientWithBaseURL(token, baseURL, httpClient),
	}
}

func (c *Client) ListRuns(owner, repo string, opts ListRunsOptions) (RunListResponse, error) {
	query := url.Values{}
	setString(query, "event", opts.Event)
	setString(query, "status", opts.Status)
	setString(query, "branch", opts.Branch)
	setString(query, "executor", opts.Executor)
	setString(query, "pull_request_id", opts.PullRequestID)
	setString(query, "workflow_id", opts.WorkflowID)
	setString(query, "workflow_name", opts.WorkflowName)
	setPositiveInt(query, "page", opts.Page)
	setPositiveInt(query, "per_page", opts.PerPage)
	setPositiveInt64(query, "startTime", opts.StartTime)
	setPositiveInt64(query, "endTime", opts.EndTime)

	var result RunListResponse
	path := repositoryPath(owner, repo) + "/actions/runs" + encodeQuery(query)
	if err := c.getJSON("list workflow runs", path, &result); err != nil {
		return RunListResponse{}, err
	}
	return result, nil
}

func (c *Client) GetRun(owner, repo, runID string) (Run, error) {
	var result Run
	path := repositoryPath(owner, repo) + "/actions/runs/" + url.PathEscape(runID)
	if err := c.getJSON("get workflow run", path, &result); err != nil {
		return Run{}, err
	}
	return result, nil
}

func (c *Client) ListJobs(owner, repo, runID string) (JobListResponse, error) {
	var result JobListResponse
	path := repositoryPath(owner, repo) + "/actions/runs/" + url.PathEscape(runID) + "/jobs"
	if err := c.getJSON("list workflow run jobs", path, &result); err != nil {
		return JobListResponse{}, err
	}
	return result, nil
}

func (c *Client) GetJob(owner, repo, runID, jobID string) (Job, error) {
	var result Job
	path := repositoryPath(owner, repo) + "/actions/runs/" + url.PathEscape(runID) + "/jobs/" + url.PathEscape(jobID)
	if err := c.getJSON("get workflow run job", path, &result); err != nil {
		// AtomGit currently returns 200 with an empty body when a job does not
		// exist, so translate the resulting decoder EOF into a useful error.
		if errors.Is(err, io.EOF) {
			return Job{}, fmt.Errorf("get workflow run job: job %s not found (API returned an empty response)", jobID)
		}
		return Job{}, err
	}
	return result, nil
}

func (c *Client) DownloadJobLog(owner, repo, runID, jobID string) (*http.Response, error) {
	path := repositoryPath(owner, repo) + "/actions/runs/" + url.PathEscape(runID) + "/jobs/" + url.PathEscape(jobID) + "/download_log"
	return c.download("download workflow run job log", path)
}

func (c *Client) ListRunArtifacts(owner, repo, runID string, opts ListArtifactsOptions) (ArtifactListResponse, error) {
	query := artifactQuery(opts)
	var result ArtifactListResponse
	path := repositoryPath(owner, repo) + "/actions/runs/" + url.PathEscape(runID) + "/artifacts" + encodeQuery(query)
	if err := c.getJSON("list workflow run artifacts", path, &result); err != nil {
		return ArtifactListResponse{}, err
	}
	return result, nil
}

func (c *Client) ListArtifacts(owner, repo string, opts ListArtifactsOptions) (ArtifactListResponse, error) {
	query := artifactQuery(opts)
	var result ArtifactListResponse
	path := repositoryPath(owner, repo) + "/actions/artifacts" + encodeQuery(query)
	if err := c.getJSON("list repository artifacts", path, &result); err != nil {
		return ArtifactListResponse{}, err
	}
	return result, nil
}

func (c *Client) GetArtifact(owner, repo, artifactID string) (Artifact, error) {
	var result Artifact
	path := repositoryPath(owner, repo) + "/actions/artifacts/" + url.PathEscape(artifactID)
	if err := c.getJSON("get artifact", path, &result); err != nil {
		return Artifact{}, err
	}
	return result, nil
}

func (c *Client) DownloadArtifact(owner, repo, artifactID string) (*http.Response, error) {
	path := repositoryPath(owner, repo) + "/actions/artifacts/" + url.PathEscape(artifactID) + "/zip"
	return c.download("download artifact", path)
}

func (c *Client) ListWorkflows(owner, repo string, opts ListWorkflowsOptions) (WorkflowListResponse, error) {
	query := url.Values{}
	setPositiveInt(query, "page", opts.Page)
	setPositiveInt(query, "per_page", opts.PerPage)

	var result WorkflowListResponse
	path := repositoryPath(owner, repo) + "/actions/workflows" + encodeQuery(query)
	if err := c.getJSON("list repository workflows", path, &result); err != nil {
		return WorkflowListResponse{}, err
	}
	return result, nil
}

func (c *Client) CreateWorkflowDispatch(owner, repo, workflowID string, payload WorkflowDispatchPayload) error {
	path := repositoryPath(owner, repo) + "/actions/workflows/" + url.PathEscape(workflowID) + "/dispatches"
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("create workflow dispatch: marshal payload: %w", err)
	}

	resp, err := c.client.DoRequestRawWithBody(http.MethodPost, path, bodyBytes, "application/json", "application/json")
	if err != nil {
		return fmt.Errorf("create workflow dispatch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return responseError("create workflow dispatch", resp)
	}
	return nil
}

func (c *Client) getJSON(operation, path string, result interface{}) error {
	resp, err := c.client.DoRequestRawWithAccept(http.MethodGet, path, "application/json")
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return responseError(operation, resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("%s: decode response: %w", operation, err)
	}
	return nil
}

func (c *Client) download(operation, path string) (*http.Response, error) {
	resp, err := c.client.DoRequestRawWithAccept(http.MethodGet, path, "*/*")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	if resp.StatusCode != http.StatusOK {
		err := responseError(operation, resp)
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

func responseError(operation string, resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	message := strings.TrimSpace(string(body))
	if len(body) > 0 {
		var details struct {
			ErrorMessage string `json:"error_message"`
			Message      string `json:"message"`
			Error        string `json:"error"`
		}
		if json.Unmarshal(body, &details) == nil {
			switch {
			case details.ErrorMessage != "":
				message = details.ErrorMessage
			case details.Message != "":
				message = details.Message
			case details.Error != "":
				message = details.Error
			}
		}
	}

	return &HTTPError{
		Operation:  operation,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Message:    message,
		RetryAfter: resp.Header.Get("Retry-After"),
		ReadError:  readErr,
	}
}

func repositoryPath(owner, repo string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}

func artifactQuery(opts ListArtifactsOptions) url.Values {
	query := url.Values{}
	setString(query, "name", opts.Name)
	setString(query, "sort", opts.Sort)
	setString(query, "direction", opts.Direction)
	setPositiveInt(query, "page", opts.Page)
	setPositiveInt(query, "per_page", opts.PerPage)
	return query
}

func setString(values url.Values, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		values.Set(key, value)
	}
}

func setPositiveInt(values url.Values, key string, value int) {
	if value > 0 {
		values.Set(key, strconv.Itoa(value))
	}
}

func setPositiveInt64(values url.Values, key string, value int64) {
	if value > 0 {
		values.Set(key, strconv.FormatInt(value, 10))
	}
}

func encodeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}
