package config

import (
	"encoding/json"
	"errors"
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
	for _, tc := range []struct{ name, body string }{
		{"unsupported version", `{"version":999}`},
		{"missing active account", `{"version":2,"active":"missing","accounts":[{"user":"alice","access_token":"fixture"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseCredentialStore([]byte(tc.body))
			if _, format := errors.AsType[*CredentialFormatError](err); err == nil || format {
				t.Fatalf("error = %v, want semantic validation failure", err)
			}
		})
	}
}
