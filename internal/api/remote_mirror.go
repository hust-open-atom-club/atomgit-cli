package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const remoteMirrorPageSize = 100

var (
	remoteMirrorURLPattern      = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s"'<>]+`)
	remoteMirrorUserinfoPattern = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*://)[^/?#@]*@`)
	remoteMirrorSCPPattern      = regexp.MustCompile(`\b[^\s/@]+@([A-Za-z0-9.-]+:[^\s"'<>]+)`)
)

// RemoteMirror is the common read-only shape returned by AtomGit's push
// remote-mirror list and repository remote-mirror state endpoints. Pointers
// preserve the distinction between a returned zero value and an absent field.
type RemoteMirror struct {
	ID                     *int64  `json:"id,omitempty"`
	ProjectID              *int64  `json:"project_id,omitempty"`
	UpdateStatus           *string `json:"update_status,omitempty"`
	LastUpdateAt           *string `json:"last_update_at,omitempty"`
	URL                    *string `json:"url,omitempty"`
	LastSuccessfulUpdateAt *string `json:"last_successful_update_at,omitempty"`
	NumberOfFailures       *int    `json:"number_of_failures,omitempty"`
	MirroringEnabled       *bool   `json:"mirroring_enabled,omitempty"`
	IsPrivate              *bool   `json:"is_private,omitempty"`
	LastError              *string `json:"last_error,omitempty"`
	Message                *string `json:"message,omitempty"`
	Force                  *bool   `json:"force,omitempty"`
	CreatedAt              *string `json:"created_at,omitempty"`
	UpdatedAt              *string `json:"updated_at,omitempty"`
}

// ListRepositoryPushRemoteMirrors retrieves at most limit configured push
// mirrors. The endpoint follows AtomGit's standard page/per_page convention.
func ListRepositoryPushRemoteMirrors(client *Client, owner, repo string, limit int) ([]RemoteMirror, error) {
	owner = url.PathEscape(owner)
	repo = url.PathEscape(repo)
	mirrors, err := GetPaginatedWithPageSize[RemoteMirror](client, limit, remoteMirrorPageSize, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/push_remote_mirrors?page=%d&per_page=%d", owner, repo, page, perPage)
	})
	if err != nil {
		return nil, redactRemoteMirrorError(err)
	}
	for index := range mirrors {
		mirrors[index] = sanitizeRemoteMirror(mirrors[index])
	}
	return mirrors, nil
}

// GetRepositoryRemoteMirror retrieves the repository's remote-mirror state.
func GetRepositoryRemoteMirror(client *Client, owner, repo string) (RemoteMirror, error) {
	var mirror RemoteMirror
	path := fmt.Sprintf("/repos/%s/%s/repo_remote_mirror", url.PathEscape(owner), url.PathEscape(repo))
	if err := client.Get(path, &mirror); err != nil {
		return RemoteMirror{}, redactRemoteMirrorError(err)
	}
	return sanitizeRemoteMirror(mirror), nil
}

func sanitizeRemoteMirror(mirror RemoteMirror) RemoteMirror {
	if mirror.URL != nil {
		value := sanitizeRemoteMirrorURL(*mirror.URL)
		mirror.URL = &value
	}
	for _, value := range []**string{
		&mirror.UpdateStatus,
		&mirror.LastUpdateAt,
		&mirror.LastSuccessfulUpdateAt,
		&mirror.LastError,
		&mirror.Message,
		&mirror.CreatedAt,
		&mirror.UpdatedAt,
	} {
		if *value != nil {
			sanitized := sanitizeRemoteMirrorText(**value)
			*value = &sanitized
		}
	}
	return mirror
}

func sanitizeRemoteMirrorURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	parsed, err := url.Parse(value)
	if err == nil && (parsed.Scheme != "" || parsed.Host != "") {
		parsed.User = nil
		parsed.RawQuery = ""
		parsed.ForceQuery = false
		parsed.Fragment = ""
		return parsed.String()
	}

	value = remoteMirrorUserinfoPattern.ReplaceAllString(value, "$1")
	if suffix := strings.IndexAny(value, "?#"); suffix >= 0 {
		value = value[:suffix]
	}
	return remoteMirrorSCPPattern.ReplaceAllString(value, "$1")
}

func sanitizeRemoteMirrorText(value string) string {
	value = strings.ReplaceAll(value, `\/`, "/")
	value = remoteMirrorURLPattern.ReplaceAllStringFunc(value, sanitizeRemoteMirrorURL)
	value = remoteMirrorSCPPattern.ReplaceAllString(value, "$1")
	return redactCredentials(value)
}

type remoteMirrorError struct {
	cause error
}

func redactRemoteMirrorError(err error) error {
	if err == nil {
		return nil
	}
	return &remoteMirrorError{cause: err}
}

func (err *remoteMirrorError) Error() string {
	return sanitizeRemoteMirrorText(err.cause.Error())
}

func (err *remoteMirrorError) Unwrap() error {
	return err.cause
}
