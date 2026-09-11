// Package apicontract provides offline fixtures and bounded response validation
// for API contract tests. It is not used by the ag runtime.
package apicontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const MaxBodyBytes = 1 << 20

// Shape describes the fields consumed by a client. Additional fields are
// allowed; removing a required field or changing its type is a breaking change.
type Shape struct {
	Types      []string         `json:"types"`
	Required   []string         `json:"required,omitempty"`
	Properties map[string]Shape `json:"properties,omitempty"`
	Items      *Shape           `json:"items,omitempty"`
}

type Example struct {
	Name    string            `json:"name"`
	Query   map[string]string `json:"query,omitempty"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"` // omitted means zero bytes, unlike JSON null
}

type Fixture struct {
	Name        string            `json:"name"`
	Source      string            `json:"source"` // synthetic or sanitized capture; never implies live verification
	Version     string            `json:"version"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Statuses    []int             `json:"statuses"`
	BodyPolicy  string            `json:"body_policy"` // required, optional, empty
	Shape       *Shape            `json:"shape,omitempty"`
	HeaderTypes map[string]string `json:"header_types,omitempty"` // optional response headers
	Examples    []Example         `json:"examples"`
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func Load(files fs.FS, pattern string) ([]Fixture, error) {
	names, err := fs.Glob(files, pattern)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, errors.New("no contract fixtures found")
	}
	fixtures := make([]Fixture, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		data, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, err
		}
		var f Fixture
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if err := d.Decode(&f); err != nil {
			return nil, fmt.Errorf("%s: invalid fixture JSON", name)
		}
		if d.Decode(new(any)) != io.EOF {
			return nil, fmt.Errorf("%s: trailing fixture data", name)
		}
		if err := f.check(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("%s: duplicate fixture name", name)
		}
		seen[f.Name] = true
		fixtures = append(fixtures, f)
	}
	return fixtures, nil
}

func (f Fixture) check() error {
	if !identifier.MatchString(f.Name) || strings.TrimSpace(f.Source) == "" {
		return errors.New("name and source are required")
	}
	if f.Version != "v5" && f.Version != "v8" {
		return errors.New("unsupported API version")
	}
	if !slices.Contains([]string{"GET", "POST", "PATCH", "PUT", "DELETE"}, f.Method) {
		return errors.New("unsupported method")
	}
	if _, err := f.URL("fixture-owner", "fixture-repo", nil); err != nil {
		return err
	}
	if len(f.Statuses) == 0 {
		return errors.New("missing success statuses")
	}
	for _, s := range f.Statuses {
		if s < 200 || s >= 300 {
			return errors.New("invalid success status")
		}
	}
	if !slices.Contains([]string{"required", "optional", "empty"}, f.BodyPolicy) {
		return errors.New("invalid body policy")
	}
	if f.BodyPolicy != "empty" && f.Shape == nil {
		return errors.New("non-empty body requires a shape")
	}
	if f.Shape != nil {
		if err := f.Shape.check(); err != nil {
			return err
		}
	}
	for k, v := range f.HeaderTypes {
		if !slices.Contains([]string{"X-Page", "X-Per-Page", "X-Total", "X-Total-Pages", "Retry-After", "Link"}, k) || !slices.Contains([]string{"integer", "string"}, v) {
			return errors.New("invalid response header contract")
		}
	}
	if len(f.Examples) == 0 {
		return errors.New("missing examples")
	}
	names := map[string]bool{}
	for _, ex := range f.Examples {
		if !identifier.MatchString(ex.Name) || names[ex.Name] {
			return errors.New("invalid or duplicate example name")
		}
		names[ex.Name] = true
		if _, err := f.URL("fixture-owner", "fixture-repo", ex.Query); err != nil {
			return err
		}
		h := http.Header{}
		for k, v := range ex.Headers {
			if k != "Content-Type" {
				if _, ok := f.HeaderTypes[k]; !ok {
					return errors.New("unlisted fixture header")
				}
			}
			h.Set(k, v)
		}
		if err := f.Validate(ex.Status, h, ex.Body); err != nil {
			return fmt.Errorf("example %s: %w", ex.Name, err)
		}
	}
	return nil
}

// URL only permits repository-scoped paths on the official API origin and
// positive pagination parameters. It never accepts a caller-supplied origin.
func (f Fixture) URL(owner, repo string, query map[string]string) (string, error) {
	if f.Version != "v5" && f.Version != "v8" {
		return "", errors.New("unsupported API version")
	}
	const root = "/repos/{owner}/{repo}"
	if f.Path != root && !strings.HasPrefix(f.Path, root+"/") {
		return "", errors.New("expected repository-scoped path pattern")
	}
	if !identifier.MatchString(owner) || !identifier.MatchString(repo) {
		return "", errors.New("invalid test repository; expected owner/repo")
	}
	path := strings.NewReplacer("{owner}", owner, "{repo}", repo).Replace(f.Path)
	if strings.ContainsAny(path, "{}?%#\\") {
		return "", errors.New("invalid path pattern")
	}
	for _, part := range strings.Split(path, "/") {
		if part == "." || part == ".." {
			return "", errors.New("invalid path segment")
		}
	}
	q := url.Values{}
	for k, v := range query {
		n, err := strconv.Atoi(v)
		if (k != "page" && k != "per_page") || err != nil || n <= 0 || n > 100 {
			return "", errors.New("invalid pagination query")
		}
		q.Set(k, v)
	}
	return (&url.URL{Scheme: "https", Host: "api.atomgit.com", Path: "/api/" + f.Version + path, RawQuery: q.Encode()}).String(), nil
}

// Validate deliberately never includes response values in errors. It can be
// used on private live responses without logging payloads or credential fields.
func (f Fixture) Validate(status int, headers http.Header, body []byte) error {
	if !slices.Contains(f.Statuses, status) {
		return fmt.Errorf("unexpected HTTP status %d", status)
	}
	if len(body) > MaxBodyBytes {
		return errors.New("response exceeds contract size limit")
	}
	for k, kind := range f.HeaderTypes {
		if value := headers.Get(k); value != "" && kind == "integer" {
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil || n < 0 {
				return fmt.Errorf("header %s: expected non-negative integer", k)
			}
		}
	}
	empty := len(bytes.TrimSpace(body)) == 0
	if empty {
		if f.BodyPolicy == "required" {
			return errors.New("required response body is empty")
		}
		return nil
	}
	if f.BodyPolicy == "empty" {
		return errors.New("expected an empty response body")
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if d.Decode(&value) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("invalid JSON response")
	}
	if f.Shape == nil {
		return errors.New("missing response shape")
	}
	return f.Shape.validate(value, "$")
}

func (s Shape) check() error {
	if len(s.Types) == 0 {
		return errors.New("shape requires types")
	}
	for _, t := range s.Types {
		if !slices.Contains([]string{"object", "array", "string", "integer", "number", "boolean", "null"}, t) {
			return errors.New("unknown shape type")
		}
	}
	if slices.Contains(s.Types, "array") && s.Items == nil {
		return errors.New("array requires item shape")
	}
	if s.Items != nil {
		if !slices.Contains(s.Types, "array") {
			return errors.New("item shape requires array type")
		}
		if err := s.Items.check(); err != nil {
			return err
		}
	}
	if (len(s.Required) > 0 || len(s.Properties) > 0) && !slices.Contains(s.Types, "object") {
		return errors.New("properties require object type")
	}
	for _, k := range s.Required {
		if _, ok := s.Properties[k]; !ok {
			return errors.New("required field has no shape")
		}
	}
	for k, p := range s.Properties {
		if !identifier.MatchString(k) {
			return errors.New("invalid shape field name")
		}
		if err := p.check(); err != nil {
			return err
		}
	}
	return nil
}

func (s Shape) validate(v any, path string) error {
	kind := "null"
	switch x := v.(type) {
	case map[string]any:
		kind = "object"
	case []any:
		kind = "array"
	case string:
		kind = "string"
	case bool:
		kind = "boolean"
	case json.Number:
		kind = "number"
		if _, err := x.Int64(); err == nil {
			kind = "integer"
		}
	}
	if !slices.Contains(s.Types, kind) && !(kind == "integer" && slices.Contains(s.Types, "number")) {
		return fmt.Errorf("%s: incompatible JSON type", path)
	}
	switch x := v.(type) {
	case map[string]any:
		for _, k := range s.Required {
			if _, ok := x[k]; !ok {
				return fmt.Errorf("%s.%s: required field missing", path, k)
			}
		}
		keys := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if value, ok := x[k]; ok {
				if err := s.Properties[k].validate(value, path+"."+k); err != nil {
					return err
				}
			}
		}
	case []any:
		if s.Items == nil {
			return errors.New("missing array item shape")
		}
		for _, value := range x {
			if err := s.Items.validate(value, path+"[]"); err != nil {
				return err
			}
		}
	}
	return nil
}
