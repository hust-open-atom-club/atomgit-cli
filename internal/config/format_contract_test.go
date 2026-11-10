package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestCredentialDecoderExposesFormatFacts(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		offset    int64
		fieldType bool
	}{
		{"syntax", `{secret`, 2, false},
		{"envelope field type", `{"version":"private-version"}`, 0, true},
		{"legacy field type", `{"access_token":123,"user":"alice"}`, 0, true},
		{"account field type", `{"version":2,"active":"alice","accounts":[{"access_token":123,"user":"alice"}]}`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseCredentialStore([]byte(tc.body))
			format, ok := errors.AsType[*CredentialFormatError](err)
			if !ok || !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("error = %v, want credential format failure", err)
			}
			if format.Offset != tc.offset || format.InvalidFieldType != tc.fieldType {
				t.Fatalf("format facts = %#v, want offset %d, field type %t", format, tc.offset, tc.fieldType)
			}
			_, syntax := errors.AsType[*json.SyntaxError](err)
			_, fieldType := errors.AsType[*json.UnmarshalTypeError](err)
			if syntax != (tc.offset > 0) || fieldType != tc.fieldType {
				t.Fatalf("decoder cause lost: %v", err)
			}
		})
	}
}

func TestCredentialValidationIsNotDecodingFailure(t *testing.T) {
	for _, tc := range []struct{ name, body, reason string }{
		{"unsupported version", `{"version":999}`, "unsupported credential store version 999"},
		{"missing store version", `{"accounts":[]}`, "credential store version is required"},
		{"no accounts", `{"version":2,"accounts":[]}`, "token file has no accounts"},
		{"invalid account", `{"version":2,"active":"alice","accounts":[{"user":"alice"}]}`, "invalid account 0: access_token is empty"},
		{"duplicate account", `{"version":2,"active":"alice","accounts":[{"user":"Alice","access_token":"fixture"},{"user":"alice","access_token":"fixture"}]}`, "duplicate account alice"},
		{"missing active account", `{"version":2,"active":"missing","accounts":[{"user":"alice","access_token":"fixture"}]}`, `active account "missing" was not found`},
		{"invalid legacy account", `{"user":"alice"}`, "token file has empty access_token or invalid account: access_token is empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, legacy, err := parseCredentialStore([]byte(tc.body))
			if err == nil || store != nil || legacy {
				t.Fatalf("expected rejected credential data, error = %v", err)
			}
			wrapped := fmt.Errorf("caller: %w", err)
			if !errors.Is(wrapped, ErrInvalidCredentials) || errors.Is(wrapped, ErrTokenNotFound) || errors.Is(wrapped, ErrNotAuthenticated) || errors.Is(wrapped, ErrCredentialAccess) {
				t.Fatalf("error = %v, want invalid credential data", wrapped)
			}
			if _, format := errors.AsType[*CredentialFormatError](wrapped); format {
				t.Fatalf("validation failure was classified as decoding failure: %v", wrapped)
			}
			validation, ok := errors.AsType[*CredentialValidationError](wrapped)
			if !ok || validation.Err == nil || !strings.HasPrefix(validation.Err.Error(), tc.reason) || !errors.Is(wrapped, validation.Err) {
				t.Fatalf("validation reason was lost: %v", wrapped)
			}
		})
	}
}

func TestMissingActiveAccountIsInvalidCredentialData(t *testing.T) {
	store := &CredentialStore{Active: "missing", Accounts: []StoredCredentials{{User: "alice", AccessToken: "fixture"}}}
	account, err := store.ActiveAccount()
	if account != nil || !errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("ActiveAccount() error = %v, want invalid active reference", err)
	}
}
