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
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	State     string    `json:"state"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

type WorkflowListResponse struct {
	TotalCount int        `json:"total_count"`
	Workflows  []Workflow `json:"workflows"`
}

type WorkflowDispatchPayload struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs,omitempty"`
}
