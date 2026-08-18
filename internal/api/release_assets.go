package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// GetReleaseUploadURL fetches the external upload destination for an attachment.
func GetReleaseUploadURL(client *Client, owner, repo, tag, fileName string) (ReleaseUploadURL, error) {
	base := fmt.Sprintf(
		"/repos/%s/%s/releases/%s/upload_url",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(tag),
	)
	path := base + "?" + url.Values{"file_name": []string{fileName}}.Encode()

	var target ReleaseUploadURL
	if err := client.Get(path, &target); err != nil {
		return ReleaseUploadURL{}, err
	}
	return target, nil
}

// UploadReleaseAsset streams a PUT to an API-provided object-store target.
// It retries once only when the non-empty body has not started transferring.
func UploadReleaseAsset(ctx context.Context, client *Client, upload ReleaseUploadURL, body io.ReadSeeker) error {
	if ctx == nil {
		return fmt.Errorf("upload context is nil")
	}
	if client == nil {
		return fmt.Errorf("API client is nil")
	}
	if upload.URL == "" {
		return fmt.Errorf("upload URL is empty")
	}
	if body == nil {
		return fmt.Errorf("upload body is nil")
	}
	target, err := url.Parse(upload.URL)
	if err != nil || !target.IsAbs() || !strings.EqualFold(target.Scheme, "https") || target.Hostname() == "" {
		return fmt.Errorf("invalid upload URL: must be an absolute HTTPS URL with a non-empty host")
	}

	httpClient := streamingHTTPClient(client)
	// A signed object-store upload target must be used exactly as returned by
	// AtomGit. Following a redirect could change PUT to GET (301/302/303),
	// replay the body outside the guarded retry path (307/308), or forward
	// upload-specific headers to another origin.
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const maxAttempts = 2
	const retryDelay = 200 * time.Millisecond
	const requestBodyCloseTimeout = 5 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Rewind the body to its start before re-sending.
			if _, err := body.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("rewind upload body before retry: %w", err)
			}
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return fmt.Errorf("upload canceled before retry: %w", ctx.Err())
			}
		}

		// A streaming reader still needs an explicit Content-Length.
		size, err := body.Seek(0, io.SeekEnd)
		if err != nil {
			return fmt.Errorf("measure upload size: %w", err)
		}
		if _, err := body.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("rewind upload body after measure: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.URL, body)
		if err != nil {
			return fmt.Errorf("build upload request: %w", err)
		}
		// Do not inject AtomGit authentication into the external request.
		for key, value := range upload.Headers {
			req.Header.Set(key, value)
		}
		req.ContentLength = size
		// Track whether the transport started consuming the request body. A
		// transport error is only safe to replay when zero bytes were read.
		trackedBody := &readTrackingReadSeeker{rs: body}
		// The transport closes req.Body after each Do. When body is an *os.File,
		// keep that Close from propagating so a safe retry can still Seek(0),
		// while retaining a signal that the transport has finished with it.
		requestBody := newNoopCloseReadSeeker(trackedBody)
		req.Body = requestBody

		resp, err := httpClient.Do(req)
		if err != nil {
			// Close any partial response body the transport may have left.
			if resp != nil {
				resp.Body.Close()
			}
			if closeErr := requestBody.waitForClose(requestBodyCloseTimeout); closeErr != nil {
				return fmt.Errorf("wait for upload request body to close after transport error: %w", closeErr)
			}
			if ctx.Err() != nil {
				return fmt.Errorf("upload canceled: %w", ctx.Err())
			}
			if size == 0 {
				return fmt.Errorf("zero-byte upload interrupted; remote state may be unknown: %w", err)
			}
			if trackedBody.bytesRead > 0 {
				return fmt.Errorf("upload interrupted after transfer started; remote state may be unknown: %w", err)
			}
			lastErr = err
			continue
		}

		if resp.StatusCode/100 == 2 {
			resp.Body.Close()
			if closeErr := requestBody.waitForClose(requestBodyCloseTimeout); closeErr != nil {
				return fmt.Errorf("wait for upload request body to close after success: %w", closeErr)
			}
			return nil
		}

		// HTTP failure: read the body for the message, do not retry.
		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if closeErr := requestBody.waitForClose(requestBodyCloseTimeout); closeErr != nil {
			return fmt.Errorf("upload failed: %s; wait for request body to close: %w", resp.Status, closeErr)
		}
		return fmt.Errorf("upload failed: %s - %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	return fmt.Errorf("upload failed after safe retry: %w", lastErr)
}

// readTrackingReadSeeker records how many body bytes a transport consumed.
// A zero-byte transport failure is safe to retry; any positive value makes the
// remote upload result ambiguous and therefore must not be replayed blindly.
type readTrackingReadSeeker struct {
	rs        io.ReadSeeker
	bytesRead int64
}

func (r *readTrackingReadSeeker) Read(p []byte) (int, error) {
	n, err := r.rs.Read(p)
	r.bytesRead += int64(n)
	return n, err
}

func (r *readTrackingReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return r.rs.Seek(offset, whence)
}

// noopCloseReadSeeker wraps an io.ReadSeeker so transport Close calls do not
// close the underlying reader. Close instead signals waitForClose, allowing the
// caller to obey RoundTripper's requirement not to reuse a body until the prior
// request has closed it.
type noopCloseReadSeeker struct {
	rs        io.ReadSeeker
	closed    chan struct{}
	closeOnce sync.Once
}

func newNoopCloseReadSeeker(rs io.ReadSeeker) *noopCloseReadSeeker {
	return &noopCloseReadSeeker{rs: rs, closed: make(chan struct{})}
}

func (n *noopCloseReadSeeker) Read(p []byte) (int, error) { return n.rs.Read(p) }
func (n *noopCloseReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return n.rs.Seek(offset, whence)
}
func (n *noopCloseReadSeeker) Close() error {
	n.closeOnce.Do(func() {
		close(n.closed)
	})
	return nil
}

func (n *noopCloseReadSeeker) waitForClose(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-n.closed:
		return nil
	case <-timer.C:
		return fmt.Errorf("timed out after %s", timeout)
	}
}

// DeleteReleaseAttachment removes an uploaded attachment from a release.
func DeleteReleaseAttachment(client *Client, owner, repo, tag string, attachmentID int64) error {
	if attachmentID <= 0 {
		return fmt.Errorf("invalid attachment id: %d (must be positive)", attachmentID)
	}

	path := fmt.Sprintf(
		"/repos/%s/%s/releases/%s/attach_files/%d",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(tag),
		attachmentID,
	)

	resp, err := client.doRequestWithPolicy(
		client.httpClient,
		http.MethodDelete,
		path,
		nil,
		"",
		"application/json",
		false,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete attachment failed: %s - %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

// DownloadReleaseAttachment returns a streamed attachment body owned by the caller.
func DownloadReleaseAttachment(ctx context.Context, client *Client, owner, repo, tag, fileName string) (io.ReadCloser, error) {
	if ctx == nil {
		return nil, fmt.Errorf("download context is nil")
	}
	path := fmt.Sprintf(
		"/repos/%s/%s/releases/%s/attach_files/%s/download",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(tag),
		url.PathEscape(fileName),
	)

	resp, err := client.doRequestWithPolicyContext(
		ctx,
		streamingHTTPClient(client),
		http.MethodGet,
		path,
		nil,
		"",
		"*/*",
		true,
	)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s - %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return resp.Body, nil
}
