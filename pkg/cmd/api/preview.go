package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"slices"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const previewRedacted = "[redacted]"
const previewFieldLimit = 50

// The preview contains only fixed vocabulary, counts, and JSON types. Never
// copy arbitrary request values into it, even when their keys look harmless.
type requestPreview struct {
	SchemaVersion int               `json:"schemaVersion"`
	DryRun        bool              `json:"dryRun"`
	Executed      bool              `json:"executed"`
	Method        string            `json:"method"`
	APIVersion    string            `json:"apiVersion"`
	Host          string            `json:"host"`
	BasePath      string            `json:"basePath"`
	Path          string            `json:"path"`
	Query         []previewQuery    `json:"query"`
	Accept        string            `json:"accept"`
	Body          previewBody       `json:"body"`
	Pagination    previewPagination `json:"pagination"`
}

type previewQuery struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type previewBody struct {
	Source      string         `json:"source"`
	ContentType string         `json:"contentType"`
	ByteLength  int            `json:"byteLength"`
	Type        string         `json:"type"`
	Count       int            `json:"count"`
	Fields      []previewField `json:"fields"`
	Truncated   bool           `json:"truncated"`
}

type previewField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type previewPagination struct {
	Enabled   bool    `json:"enabled"`
	FirstPage *string `json:"firstPage"`
	PerPage   *string `json:"perPage"`
	Strategy  string  `json:"strategy"`
}

// preparationError preserves real-mode diagnostics and error identity, but
// prevents user-controlled method, endpoint, field, file, or reader errors
// from echoing secrets during a dry run (including with --raw-output).
func preparationError(opts options, safeMessage string, cause error) error {
	if opts.dryRun {
		return &redactedError{message: safeMessage, cause: cause}
	}
	return cause
}

func writePreview(out io.Writer, request preparedRequest) error {
	base, err := url.Parse(request.baseURL)
	if err != nil {
		return err
	}
	endpoint, err := url.Parse(request.path)
	if err != nil {
		return err
	}
	preview := requestPreview{
		SchemaVersion: 1, DryRun: true, Executed: false,
		Method: request.method, APIVersion: strings.TrimPrefix(base.Path, "/api/"),
		Host: base.Host, BasePath: base.Path, Path: previewPath(endpoint.EscapedPath()),
		Query: make([]previewQuery, 0), Accept: previewRedacted,
		Body: describeBody(request), Pagination: previewPagination{Strategy: "none"},
	}
	if request.accept == "application/json" {
		preview.Accept = "application/json"
	}
	query := endpoint.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		preview.Query = append(preview.Query, previewQuery{Name: previewName(key), Type: "string", Count: len(query[key])})
	}
	if request.pagination != nil {
		// Pagination values are request inputs too. Keep the strategy visible,
		// but never copy numeric inputs into an otherwise redacted preview.
		redacted := previewRedacted
		preview.Pagination = previewPagination{
			Enabled: true, FirstPage: &redacted, PerPage: &redacted,
			Strategy: "total_page-or-short-array",
		}
	}
	return cmdutil.WriteJSON(out, preview)
}

// These are display allowlists, not endpoint validation or routing rules.
// Unknown names, identifiers and encoded segments are intentionally hidden.
func previewPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		switch segment {
		case "", "user", "users", "repos", "orgs", "issues", "pulls", "comments", "releases", "tags", "branches", "labels", "milestones", "hooks", "keys", "contents", "notifications", "members", "collaborators", "commits":
		default:
			segments[i] = previewRedacted
		}
	}
	return strings.Join(segments, "/")
}

func previewName(name string) string {
	switch name {
	case "title", "body", "name", "description", "state", "labels", "page", "per_page", "q", "sort", "direction", "access_token", "refresh_token", "token", "password", "client_id", "client_secret", "code", "secret", "authorization", "webhook_secret":
		return name
	default:
		return previewRedacted
	}
}

func describeBody(request preparedRequest) previewBody {
	result := previewBody{
		Source: request.bodySource, ContentType: request.contentType,
		ByteLength: len(request.body), Type: "absent", Fields: make([]previewField, 0),
	}
	if request.bodySource == "none" {
		return result
	}
	result.Type = "opaque"
	if !json.Valid(request.body) {
		return result
	}
	result.Type = jsonType(request.body)
	switch result.Type {
	case "object":
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(request.body, &fields) // json.Valid checked the complete document.
		result.Count = len(fields)
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		result.Truncated = len(keys) > previewFieldLimit
		for _, key := range keys[:min(len(keys), previewFieldLimit)] {
			result.Fields = append(result.Fields, previewField{Name: previewName(key), Type: jsonType(fields[key])})
		}
	case "array":
		var elements []json.RawMessage
		_ = json.Unmarshal(request.body, &elements)
		result.Count = len(elements)
	}
	return result
}

func jsonType(value []byte) string {
	value = bytes.TrimSpace(value)
	switch value[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}
