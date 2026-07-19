package cmdutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitEditorCommand(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    []string
		wantErr string
	}{
		{name: "command and argument", value: "code --wait", want: []string{"code", "--wait"}},
		{name: "quoted executable", value: `"/Applications/Visual Studio Code.app/Contents/MacOS/Electron" --wait`, want: []string{"/Applications/Visual Studio Code.app/Contents/MacOS/Electron", "--wait"}},
		{name: "quoted argument", value: `editor --option='two words'`, want: []string{"editor", "--option=two words"}},
		{name: "windows path", value: `"C:\Program Files\Editor\editor.exe" --wait`, want: []string{`C:\Program Files\Editor\editor.exe`, "--wait"}},
		{name: "unterminated quote", value: `editor "unfinished`, wantErr: "unterminated quote"},
		{name: "trailing escape", value: `editor \`, wantErr: "trailing escape"},
		{name: "empty", value: "   ", wantErr: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitEditorCommand(tt.value)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitEditorCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEditTextUsesPrivateFileAndConfiguredEditor(t *testing.T) {
	lookupEnv := func(name string) (string, bool) {
		switch name {
		case "VISUAL":
			return `"test editor" --wait`, true
		case "EDITOR":
			return "ignored", true
		default:
			return "", false
		}
	}

	runner := func(name string, args []string, _ io.Reader, _, _ io.Writer) error {
		if name != "test editor" {
			return fmt.Errorf("editor name = %q", name)
		}
		if len(args) != 2 || args[0] != "--wait" {
			return fmt.Errorf("editor args = %#v", args)
		}

		path := args[1]
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode().Perm() != 0o600 {
			return fmt.Errorf("file mode = %o", info.Mode().Perm())
		}
		parent, err := os.Stat(filepath.Dir(path))
		if err != nil {
			return err
		}
		if parent.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("directory mode = %o", parent.Mode().Perm())
		}

		initial, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(initial) != "draft" {
			return fmt.Errorf("initial body = %q", initial)
		}
		return os.WriteFile(path, []byte("edited body\n"), 0o600)
	}

	body, err := editText(strings.NewReader(""), io.Discard, io.Discard, "draft", lookupEnv, runner)
	if err != nil {
		t.Fatal(err)
	}
	if body != "edited body\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestEditTextRequiresConfiguredEditor(t *testing.T) {
	_, err := editText(strings.NewReader(""), io.Discard, io.Discard, "", func(string) (string, bool) {
		return "", false
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "no editor configured") {
		t.Fatalf("error = %v", err)
	}
}

func TestEditTextReportsEditorFailure(t *testing.T) {
	_, err := editText(strings.NewReader(""), io.Discard, io.Discard, "", func(name string) (string, bool) {
		if name == "EDITOR" {
			return "editor", true
		}
		return "", false
	}, func(string, []string, io.Reader, io.Writer, io.Writer) error {
		return fmt.Errorf("exit status 1")
	})
	if err == nil || !strings.Contains(err.Error(), "editor failed") {
		t.Fatalf("error = %v", err)
	}
}
