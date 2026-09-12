package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Timestamp is an AtomGit Actions timestamp in milliseconds since Unix epoch.
// The API currently returns both JSON numbers and quoted numbers, so the type
// accepts either representation as well as RFC3339 timestamps.
type Timestamp int64

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = 0
		return nil
	}

	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("decode timestamp: %w", err)
		}
		if value == "" {
			*t = 0
			return nil
		}
		milliseconds, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			*t = Timestamp(milliseconds)
			return nil
		}
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return fmt.Errorf("decode timestamp %q: %w", value, parseErr)
		}
		*t = Timestamp(parsed.UnixMilli())
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("decode timestamp: %w", err)
	}
	milliseconds, err := number.Int64()
	if err != nil {
		return fmt.Errorf("decode timestamp %q: %w", number.String(), err)
	}
	*t = Timestamp(milliseconds)
	return nil
}

func (t Timestamp) Time() time.Time {
	value := int64(t)
	if value == 0 {
		return time.Time{}
	}
	if value > -1_000_000_000_000 && value < 1_000_000_000_000 {
		return time.Unix(value, 0)
	}
	return time.UnixMilli(value)
}

type Actor struct {
	ID       string `json:"id"`
	ObjectID string `json:"object_id"`
	Login    string `json:"login"`
	Name     string `json:"name"`
}

type Run struct {
	WorkflowRunID        string    `json:"workflow_run_id"`
	WorkflowID           string    `json:"workflow_id"`
	WorkflowName         string    `json:"workflow_name"`
	FilePath             string    `json:"file_path"`
	Title                string    `json:"title"`
	Status               string    `json:"status"`
	Event                string    `json:"event"`
	RunNumber            int       `json:"run_number"`
	HeadBranch           string    `json:"head_branch"`
	HeadSHA              string    `json:"head_sha"`
	Actor                Actor     `json:"actor"`
	StartTime            Timestamp `json:"start_time"`
	EndTime              Timestamp `json:"end_time"`
	PauseTime            Timestamp `json:"pause_time"`
	ExistInDefaultBranch bool      `json:"exist_in_default_branch"`
	Stages               []Stage   `json:"stages"`
}

type Stage struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartTime Timestamp `json:"start_time"`
	EndTime   Timestamp `json:"end_time"`
	Jobs      []Job     `json:"jobs"`
}

type Job struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Identifier      string    `json:"identifier"`
	Status          string    `json:"status"`
	Message         *string   `json:"message"`
	StartTime       Timestamp `json:"start_time"`
	EndTime         Timestamp `json:"end_time"`
	ExecuteCostTime int64     `json:"execute_cost_time"`
	ExecID          string    `json:"exec_id"`
	Steps           []Step    `json:"steps"`
}

type Step struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Task      string    `json:"task"`
	Status    string    `json:"status"`
	Message   *string   `json:"message"`
	StartTime Timestamp `json:"start_time"`
	EndTime   Timestamp `json:"end_time"`
}

type Artifact struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	SizeBytes     int64     `json:"size_bytes"`
	WorkflowID    string    `json:"workflow_id"`
	WorkflowRunID string    `json:"workflow_run_id"`
	Digest        string    `json:"digest"`
	ExpiresAt     Timestamp `json:"expires_at"`
	CreatedAt     Timestamp `json:"created_at"`
	UpdatedAt     Timestamp `json:"updated_at"`
}

type RunListResponse struct {
	TotalCount   int   `json:"total_count"`
	WorkflowRuns []Run `json:"workflow_runs"`
}

type JobListResponse struct {
	TotalCount int   `json:"total_count"`
	Jobs       []Job `json:"jobs"`
}

type ArtifactListResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

type Workflow struct {
	ID        string    `json:"workflow_id"`
	Name      string    `json:"name"`
	Path      string    `json:"file_path"`
	State     string    `json:"state"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

type WorkflowListResponse struct {
	TotalCount int        `json:"total_count"`
	Workflows  []Workflow `json:"workflows"`
}

// Runner represents a host runner exposed by AtomGit Actions. Busy and
// Online are pointers because the API may omit either field for a runner
// scope that does not expose that state; callers must not infer a value.
type Runner struct {
	ID           RunnerIdentifier `json:"id"`
	Name         string           `json:"name"`
	OS           string           `json:"os,omitempty"`
	Platform     string           `json:"platform,omitempty"`
	Architecture string           `json:"architecture,omitempty"`
	Status       string           `json:"status,omitempty"`
	Busy         *bool            `json:"busy,omitempty"`
	Online       *bool            `json:"online,omitempty"`
	Labels       []RunnerLabel    `json:"labels,omitempty"`
	Scope        string           `json:"scope,omitempty"`
	RunnerType   string           `json:"runner_type,omitempty"`
	Version      string           `json:"version,omitempty"`
}

// RunnerIdentifier accepts both numeric and string IDs returned by different
// Actions deployments while keeping JSON output stable as a string.
type RunnerIdentifier string

func (id *RunnerIdentifier) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*id = ""
		return nil
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("decode runner id: %w", err)
		}
		*id = RunnerIdentifier(value)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("decode runner id: %w", err)
	}
	*id = RunnerIdentifier(number.String())
	return nil
}

// RunnerLabel normalizes the object and string label forms used by Actions
// APIs. String labels are represented with only Name populated.
type RunnerLabel struct {
	ID   RunnerIdentifier `json:"id,omitempty"`
	Name string           `json:"name"`
	Type string           `json:"type,omitempty"`
}

func (label *RunnerLabel) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*label = RunnerLabel{}
		return nil
	}
	if data[0] == '"' {
		if err := json.Unmarshal(data, &label.Name); err != nil {
			return fmt.Errorf("decode runner label: %w", err)
		}
		return nil
	}
	type alias RunnerLabel
	var value alias
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode runner label: %w", err)
	}
	*label = RunnerLabel(value)
	return nil
}

type RunnerListResponse struct {
	TotalCount int      `json:"total_count"`
	Runners    []Runner `json:"runners"`
}

type WorkflowDispatchPayload struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs,omitempty"`
}

type WorkflowValidationRequest struct {
	Base64Content string `json:"base64_content"`
}

type WorkflowValidationResponse struct {
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type DiagnosticRangePoint struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type DiagnosticRange struct {
	Start DiagnosticRangePoint `json:"start"`
	End   DiagnosticRangePoint `json:"end"`
}

type Diagnostic struct {
	Range    DiagnosticRange `json:"range"`
	Severity string          `json:"severity"`
	Message  string          `json:"message"`
}

type StepLogRequest struct {
	StepID string `json:"step_id"`
	Offset int64  `json:"offset"`
	Limit  int    `json:"limit"`
	Sort   string `json:"sort"`
}

type StepLogResponse struct {
	HasMore     bool   `json:"has_more"`
	StartOffset int64  `json:"start_offset"`
	EndOffset   int64  `json:"end_offset"`
	Log         string `json:"log"`
}
