package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

// MaxErrorExcerptBytes is the shared maximum number of response-body bytes
// retained for any AtomGit API error. One additional byte is read only to
// detect truncation; it is never included in returned error text.
const MaxErrorExcerptBytes = 4096

const errorExcerptTruncationMarker = "..."

// maxEmbeddedJSONDepth bounds recursive parsing of JSON serialized inside
// string values. Eight layers cover realistic proxy wrapping while preventing
// hostile responses from causing unbounded recursion.
const maxEmbeddedJSONDepth = 8

var (
	// bearerTokenPattern matches Authorization header values embedded in error
	// bodies so they are never surfaced to the user.
	bearerTokenPattern = regexp.MustCompile(`(?i)(\bBearer\s+)[^\s,;"'}\]]+`)

	// Credential-like response keys include common token, secret, password,
	// and authorization variants such as access_token, apiToken, and
	// client_secret.
	credentialFieldPattern = `[A-Za-z0-9_.-]*(?:token|secret|password|passwd|authorization)[A-Za-z0-9_.-]*`
	credentialFieldNameRE  = regexp.MustCompile(`(?i)^` + credentialFieldPattern + `$`)

	// quotedCredentialPattern matches complete JSON string values, including
	// escaped characters inside the value.
	quotedCredentialPattern = regexp.MustCompile(`(?i)("(?:` + credentialFieldPattern + `)"\s*:\s*")(?:\\.|[^"\\])*(")`)

	// unterminatedQuotedCredentialPattern catches a credential value cut off
	// by the bounded error excerpt before its closing quote.
	unterminatedQuotedCredentialPattern = regexp.MustCompile(`(?i)("(?:` + credentialFieldPattern + `)"\s*:\s*")(?:\\.|[^"\\])*$`)

	// decodedQuotedCredentialPattern and its unterminated variant match JSON
	// string fields without assuming the key's raw spelling. The key token is
	// decoded before classification so escapes such as access\u005ftoken cannot
	// bypass the fallback used for malformed or truncated JSON.
	decodedQuotedCredentialPattern       = regexp.MustCompile(`(?s)("(?:\\.|[^"\\])*")(\s*:\s*")((?:\\.|[^"\\])*)(")`)
	decodedUnterminatedCredentialPattern = regexp.MustCompile(`(?s)("(?:\\.|[^"\\])*")(\s*:\s*")((?:\\.|[^"\\])*)$`)

	// credentialKeyValuePattern covers non-JSON excerpts such as
	// "access_token=..." and "refresh_token: ...".
	credentialKeyValuePattern = regexp.MustCompile(`(?i)(\b(?:` + credentialFieldPattern + `)\b\s*[:=]\s*)[^,;&}\]\r\n]+`)
)

// ErrorResponseDetails contains bounded, terminal-safe, credential-redacted
// context read from a non-successful AtomGit API response.
type ErrorResponseDetails struct {
	Status     string
	Body       string
	Message    string
	RetryAfter string
	ReadError  error
}

// ReadErrorResponse reads and sanitizes a single shared-size excerpt from a
// non-successful response. Message prefers the structured error_message,
// message, or error JSON field and falls back to the raw excerpt. Body retains
// the full bounded excerpt for clients whose established error shape exposes
// it directly.
func ReadErrorResponse(resp *http.Response) ErrorResponseDetails {
	var body []byte
	var readErr error
	if resp.Body != nil {
		body, readErr = io.ReadAll(io.LimitReader(resp.Body, MaxErrorExcerptBytes+1))
	}

	truncated := len(body) > MaxErrorExcerptBytes
	if truncated {
		body = body[:MaxErrorExcerptBytes]
	}

	rawBody := string(body)
	message := strings.TrimSpace(rawBody)
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

	result := ErrorResponseDetails{
		Status:     SanitizeErrorText(resp.Status),
		Body:       SanitizeErrorText(rawBody),
		Message:    SanitizeErrorText(message),
		RetryAfter: SanitizeErrorText(resp.Header.Get("Retry-After")),
		ReadError:  readErr,
	}
	if truncated {
		result.Body += errorExcerptTruncationMarker
		result.Message += errorExcerptTruncationMarker
	}
	return result
}

// SanitizeErrorText neutralizes terminal control characters and redacts
// credential-like values before API response context reaches command output.
func SanitizeErrorText(s string) string {
	s = redactJSONCredentials(s)
	s = redactEmbeddedJSONCredentials(s, maxEmbeddedJSONDepth)
	s = sanitizeAPIString(s)
	return redactCredentials(s)
}

func redactJSONCredentials(s string) string {
	decoder := json.NewDecoder(strings.NewReader(s))
	decoder.UseNumber()

	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return s
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return s
	}
	redacted, changed := redactJSONValue(value, maxEmbeddedJSONDepth)
	if !changed {
		return s
	}

	encoded, err := encodeJSONValue(redacted)
	if err != nil {
		return s
	}
	return encoded
}

func encodeJSONValue(value interface{}) (string, error) {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSuffix(encoded.String(), "\n"), nil
}

func redactJSONValue(value interface{}, embeddedDepth int) (interface{}, bool) {
	switch value := value.(type) {
	case map[string]interface{}:
		changed := false
		for key, fieldValue := range value {
			if isCredentialField(key) {
				value[key] = "<redacted>"
				changed = true
				continue
			}
			redacted, fieldChanged := redactJSONValue(fieldValue, embeddedDepth)
			if fieldChanged {
				value[key] = redacted
				changed = true
			}
		}
		return value, changed
	case []interface{}:
		changed := false
		for i, item := range value {
			redacted, itemChanged := redactJSONValue(item, embeddedDepth)
			if itemChanged {
				value[i] = redacted
				changed = true
			}
		}
		return value, changed
	case string:
		redacted := redactEmbeddedJSONCredentials(value, embeddedDepth)
		redacted = redactCredentials(redacted)
		return redacted, redacted != value
	}
	return value, false
}

func redactEmbeddedJSONCredentials(s string, depth int) string {
	if depth <= 0 {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))
	last := 0
	changed := false
	for i := 0; i < len(s); {
		if s[i] != '{' && s[i] != '[' {
			i++
			continue
		}

		decoder := json.NewDecoder(strings.NewReader(s[i:]))
		decoder.UseNumber()
		var value interface{}
		if err := decoder.Decode(&value); err != nil {
			i++
			continue
		}
		consumed := int(decoder.InputOffset())
		if consumed <= 0 {
			i++
			continue
		}

		redacted, valueChanged := redactJSONValue(value, depth-1)
		if valueChanged {
			encoded, err := encodeJSONValue(redacted)
			if err != nil {
				i += consumed
				continue
			}
			result.WriteString(s[last:i])
			result.WriteString(encoded)
			last = i + consumed
			changed = true
		}
		i += consumed
	}
	if !changed {
		return s
	}
	result.WriteString(s[last:])
	return result.String()
}

func isCredentialField(key string) bool {
	return credentialFieldNameRE.MatchString(key)
}

func sanitizeAPIString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, "\\x%02x", r)
		case unicode.IsControl(r):
			fmt.Fprintf(&b, "\\u%04x", r)
		case isUnicodeDirectionControl(r):
			fmt.Fprintf(&b, "\\u%04x", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isUnicodeDirectionControl(r rune) bool {
	return (r >= 0x202a && r <= 0x202e) ||
		(r >= 0x2066 && r <= 0x2069) ||
		r == 0x2028 || r == 0x2029 ||
		r == 0x061c ||
		r == 0x200e || r == 0x200f
}

func redactCredentials(s string) string {
	s = redactDecodedQuotedCredentials(s, decodedQuotedCredentialPattern)
	s = redactDecodedQuotedCredentials(s, decodedUnterminatedCredentialPattern)
	s = quotedCredentialPattern.ReplaceAllString(s, "${1}<redacted>${2}")
	s = unterminatedQuotedCredentialPattern.ReplaceAllString(s, "${1}<redacted>")
	s = bearerTokenPattern.ReplaceAllString(s, "${1}<redacted>")
	return credentialKeyValuePattern.ReplaceAllString(s, "${1}<redacted>")
}

func redactDecodedQuotedCredentials(s string, pattern *regexp.Regexp) string {
	matches := pattern.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))
	last := 0
	changed := false
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}
		var key string
		if err := json.Unmarshal([]byte(s[match[2]:match[3]]), &key); err != nil || !isCredentialField(key) {
			continue
		}
		result.WriteString(s[last:match[6]])
		result.WriteString("<redacted>")
		last = match[7]
		changed = true
	}
	if !changed {
		return s
	}
	result.WriteString(s[last:])
	return result.String()
}
