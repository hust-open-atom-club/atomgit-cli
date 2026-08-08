package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	internalapi "atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const maxErrorBody = 64 << 10

type options struct {
	method   string
	fields   []string
	input    string
	accept   string
	paginate bool
}

type preparedRequest struct {
	method      string
	path        string
	body        []byte
	contentType string
	accept      string
	pagination  *paginationState
}

type paginationState struct {
	page       int
	perPage    int
	totalPages int
	visited    map[string]bool
}

type redactedError struct {
	message string
	cause   error
}

func (e *redactedError) Error() string { return e.message }
func (e *redactedError) Unwrap() error { return e.cause }

func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:          "api <endpoint>",
		Short:        "Make an authenticated AtomGit API request",
		SilenceUsage: true,
		Long: `Make an authenticated request to a relative AtomGit API v5 endpoint.

GET is the default. Supported methods are GET, POST, PATCH, PUT, and DELETE.
Explicit non-GET requests may change remote resources; ag does not infer or
confirm the endpoint's effects. Redirects only retain credentials on the exact
AtomGit API origin. Paginated output is one compact JSON page per line.
Response bytes use terminal-safe output unless --raw-output is specified.`,
		Example: `  ag api /user
  ag api /repos/owner/repo/issues --field state=open
  ag api /repos/owner/repo/issues --method POST --field title='New issue'
  ag api /repos/owner/repo/issues/42 --method PATCH --input update.json
  ag api /repos/owner/repo/issues --paginate`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prepared, err := prepare(args[0], opts, cmd.InOrStdin())
			if err != nil {
				return err
			}
			return execute(cmd, f, prepared)
		},
	}
	cmd.Flags().StringVarP(&opts.method, "method", "X", http.MethodGet, "HTTP method: GET, POST, PATCH, PUT, or DELETE")
	cmd.Flags().StringArrayVarP(&opts.fields, "field", "f", nil, "Add a string field as key=value")
	cmd.Flags().StringVar(&opts.input, "input", "", "Read the raw request body from a file or - for stdin")
	cmd.Flags().StringVarP(&opts.accept, "accept", "H", "application/json", "Set the Accept request header")
	cmd.Flags().BoolVar(&opts.paginate, "paginate", false, "Request all pages and emit compact JSON pages as NDJSON")
	return cmd
}

func prepare(endpoint string, opts options, stdin io.Reader) (preparedRequest, error) {
	method := strings.ToUpper(strings.TrimSpace(opts.method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
	default:
		return preparedRequest{}, fmt.Errorf("unsupported HTTP method %q", opts.method)
	}
	path, values, err := parseEndpoint(endpoint)
	if err != nil {
		return preparedRequest{}, err
	}
	if !validHeaderValue(opts.accept) {
		return preparedRequest{}, fmt.Errorf("invalid Accept value")
	}
	if opts.input != "" && len(opts.fields) > 0 {
		return preparedRequest{}, fmt.Errorf("--input and --field cannot be used together")
	}
	if opts.paginate && method != http.MethodGet {
		return preparedRequest{}, fmt.Errorf("--paginate requires GET")
	}
	if opts.paginate && opts.input != "" {
		return preparedRequest{}, fmt.Errorf("--paginate cannot be used with --input")
	}

	fields, err := parseFields(opts.fields)
	if err != nil {
		return preparedRequest{}, err
	}
	request := preparedRequest{method: method, accept: opts.accept}
	if method == http.MethodGet {
		for _, field := range fields {
			values.Add(field[0], field[1])
		}
	} else if len(fields) > 0 {
		object := make(map[string]string, len(fields))
		for _, field := range fields {
			object[field[0]] = field[1]
		}
		request.body, err = json.Marshal(object)
		if err != nil {
			return preparedRequest{}, fmt.Errorf("encode fields: %w", err)
		}
		request.contentType = "application/json"
	}
	if opts.input != "" {
		if opts.input == "-" {
			request.body, err = io.ReadAll(stdin)
		} else {
			request.body, err = os.ReadFile(opts.input)
		}
		if err != nil {
			return preparedRequest{}, fmt.Errorf("read input %q: %w", opts.input, err)
		}
	}
	if opts.paginate {
		request.pagination, err = preparePagination(values)
		if err != nil {
			return preparedRequest{}, err
		}
	}
	request.path = path
	if encoded := values.Encode(); encoded != "" {
		request.path += "?" + encoded
	}
	return request, nil
}

func parseEndpoint(value string) (string, url.Values, error) {
	if value == "" || strings.HasPrefix(value, "//") {
		return "", nil, fmt.Errorf("endpoint must be a relative API v5 path")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", nil, fmt.Errorf("invalid endpoint: %w", err)
	}
	if parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" || parsed.Path == "" {
		return "", nil, fmt.Errorf("endpoint must be a relative API v5 path without a fragment")
	}
	unescapedPath, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", nil, fmt.Errorf("invalid endpoint path escape: %w", err)
	}
	for segment := range strings.SplitSeq(unescapedPath, "/") {
		if segment == "." || segment == ".." {
			return "", nil, fmt.Errorf("endpoint must remain within the API v5 base path")
		}
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return "", nil, fmt.Errorf("invalid endpoint query: %w", err)
	}
	return "/" + strings.TrimLeft(parsed.EscapedPath(), "/"), query, nil
}

func parseFields(values []string) ([][2]string, error) {
	fields := make([][2]string, 0, len(values))
	for _, value := range values {
		key, fieldValue, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid field %q: expected non-empty key=value", value)
		}
		fields = append(fields, [2]string{key, fieldValue})
	}
	return fields, nil
}

func validHeaderValue(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if r == '\r' || r == '\n' || r == 0 || (r < 0x20 && r != '\t') || r == 0x7f {
			return false
		}
	}
	return true
}

func preparePagination(values url.Values) (*paginationState, error) {
	state := &paginationState{page: 1, perPage: 100, visited: map[string]bool{}}
	for key, target := range map[string]*int{"page": &state.page, "per_page": &state.perPage} {
		entries := values[key]
		if len(entries) > 1 {
			return nil, fmt.Errorf("pagination query %q must occur once", key)
		}
		if len(entries) == 1 {
			parsed, err := strconv.Atoi(entries[0])
			if err != nil || parsed <= 0 {
				return nil, fmt.Errorf("pagination query %q must be a positive integer", key)
			}
			*target = parsed
		}
	}
	values.Set("page", strconv.Itoa(state.page))
	values.Set("per_page", strconv.Itoa(state.perPage))
	return state, nil
}

func execute(cmd *cobra.Command, f *cmdutil.Factory, request preparedRequest) error {
	if f == nil || f.Config == nil {
		return fmt.Errorf("configuration is unavailable")
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return redact(fmt.Errorf("not authenticated: %w", err), token)
	}
	httpClient, err := requestHTTPClient(f)
	if err != nil {
		return redact(err, token)
	}
	client := internalapi.NewClientWithHTTPClient(token, cloneRedirectSafeClient(httpClient))
	if request.pagination != nil {
		return executePagination(cmd.OutOrStdout(), client, request, token)
	}
	resp, err := client.DoRequestRawWithBody(request.method, request.path, request.body, request.contentType, request.accept)
	if err != nil {
		return redact(fmt.Errorf("request %s %s: %w", request.method, request.path, err), token)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return responseError(request.method+" "+request.path, resp, token)
	}
	if _, err := io.Copy(cmd.OutOrStdout(), resp.Body); err != nil {
		return redact(fmt.Errorf("read response: %w", err), token)
	}
	return nil
}

func requestHTTPClient(f *cmdutil.Factory) (*http.Client, error) {
	if f.HttpClient == nil {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}
	client, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("create HTTP client: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("create HTTP client: returned nil client")
	}
	return client, nil
}

func cloneRedirectSafeClient(source *http.Client) *http.Client {
	clone := *source
	previous := source.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		stripRedirectAuthorization(req, via)
		if previous != nil {
			if err := previous(req, via); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		stripRedirectAuthorization(req, via)
		return nil
	}
	return &clone
}

func stripRedirectAuthorization(req *http.Request, via []*http.Request) {
	for _, prior := range via {
		if !sameOrigin(prior.URL, req.URL) || prior.Header.Get("Authorization") == "" {
			req.Header.Del("Authorization")
			return
		}
	}
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Hostname(), b.Hostname()) && effectivePort(a) == effectivePort(b)
}

func effectivePort(value *url.URL) string {
	if value.Port() != "" {
		return value.Port()
	}
	if strings.EqualFold(value.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(value.Scheme, "http") {
		return "80"
	}
	return ""
}

func responseError(operation string, resp *http.Response, token string) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	message := strings.TrimSpace(string(body))
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
	if token != "" {
		message = strings.ReplaceAll(message, token, "[REDACTED]")
	}
	result := fmt.Errorf("%s: API request failed (%s)", operation, resp.Status)
	if message != "" {
		result = fmt.Errorf("%w: %s", result, message)
	}
	if readErr != nil {
		result = fmt.Errorf("%w: read error response: %w", result, readErr)
	}
	return result
}

// redact replaces an access token in an error message with a placeholder.
// Values shorter than 4 characters are left untouched: replacing a very short
// token would mangle ordinary words in the message while providing little
// protection, since such a short value cannot meaningfully be a credential.
func redact(err error, token string) error {
	if err == nil || token == "" || len(token) < 4 || !strings.Contains(err.Error(), token) {
		return err
	}
	return &redactedError{message: strings.ReplaceAll(err.Error(), token, "[REDACTED]"), cause: err}
}

func executePagination(out io.Writer, client *internalapi.Client, request preparedRequest, token string) error {
	state := request.pagination
	base, err := url.Parse(request.path)
	if err != nil {
		return err
	}
	for {
		query := base.Query()
		query.Set("page", strconv.Itoa(state.page))
		query.Set("per_page", strconv.Itoa(state.perPage))
		base.RawQuery = query.Encode()
		path := base.RequestURI()
		if state.visited[path] {
			return fmt.Errorf("pagination repeated request %q", path)
		}
		state.visited[path] = true
		resp, err := client.DoRequestRawWithBody(http.MethodGet, path, nil, "", request.accept)
		if err != nil {
			return redact(fmt.Errorf("request page %d: %w", state.page, err), token)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			err := responseError(fmt.Sprintf("request page %d", state.page), resp, token)
			resp.Body.Close()
			return err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return redact(fmt.Errorf("read page %d: %w", state.page, readErr), token)
		}
		var value json.RawMessage
		if json.Unmarshal(body, &value) != nil {
			return fmt.Errorf("page %d is not valid JSON", state.page)
		}
		compact := bytes.Buffer{}
		if err := json.Compact(&compact, body); err != nil {
			return fmt.Errorf("compact page %d: %w", state.page, err)
		}
		total, err := parseTotalPages(resp.Header.Get("total_page"))
		if err != nil {
			return fmt.Errorf("page %d: %w", state.page, err)
		}
		if total > 0 {
			if state.totalPages != 0 && total != state.totalPages {
				return fmt.Errorf("page %d returned inconsistent total_page", state.page)
			}
			state.totalPages = total
		}
		stop := state.totalPages > 0 && state.page >= state.totalPages
		if state.totalPages == 0 {
			var array []json.RawMessage
			if err := json.Unmarshal(body, &array); err != nil {
				return fmt.Errorf("page %d has no total_page and is not an array", state.page)
			}
			stop = len(array) < state.perPage
		}
		if _, err := compact.WriteTo(out); err != nil {
			return fmt.Errorf("write page %d: %w", state.page, err)
		}
		if _, err := io.WriteString(out, "\n"); err != nil {
			return fmt.Errorf("write page %d: %w", state.page, err)
		}
		if stop {
			return nil
		}
		state.page++
	}
}

func parseTotalPages(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	total, err := strconv.Atoi(value)
	if err != nil || total <= 0 {
		return 0, fmt.Errorf("invalid total_page header")
	}
	return total, nil
}
