package repo

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseRepositoryPassword(t *testing.T) {
	for _, fromStdin := range []bool{false, true} {
		source := "terminal"
		if fromStdin {
			source = "stdin"
		}
		t.Run(source, func(t *testing.T) {
			for _, tc := range []struct {
				name, input, want, errText string
			}{
				{name: "empty", errText: "must not be empty"},
				{name: "spaces", input: " password with spaces ", want: " password with spaces "},
				{name: "at limit", input: strings.Repeat("x", maxRepositoryPasswordBytes), want: strings.Repeat("x", maxRepositoryPasswordBytes)},
				{name: "over limit", input: strings.Repeat("x", maxRepositoryPasswordBytes+1), errText: "password from " + source + " exceeds 65536 bytes"},
				{name: "multibyte over limit", input: strings.Repeat("密", maxRepositoryPasswordBytes/3+1), errText: "exceeds 65536 bytes"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					got, err := parseRepositoryPassword([]byte(tc.input), fromStdin, "repository archive")
					if tc.errText != "" {
						if err == nil || !strings.Contains(err.Error(), tc.errText) || got != "" {
							t.Fatal("expected password validation error with no returned password")
						}
						return
					}
					if err != nil || got != tc.want {
						t.Fatal("password was rejected or changed")
					}
				})
			}
		})
	}
}

func TestParseRepositoryPasswordLineEndings(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		fromStdin         bool
	}{
		{"stdin LF", " secret \n", " secret ", true},
		{"stdin CRLF", " secret \r\n", " secret ", true},
		{"stdin removes one ending", "secret\n\n", "secret\n", true},
		{"terminal preserves bytes", "secret\r", "secret\r", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRepositoryPassword([]byte(tc.input), tc.fromStdin, "repository archive")
			if err != nil || got != tc.want {
				t.Fatal("password line endings were handled incorrectly")
			}
		})
	}
}

func TestTransferPasswordNonInteractiveContext(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	_, err := readTransferPassword(cmd, false)
	want := "organization-owned repository transfer requires an interactive password; use --yes --password-stdin for non-interactive use"
	if err == nil || err.Error() != want {
		t.Fatalf("error=%v want=%q", err, want)
	}
}
