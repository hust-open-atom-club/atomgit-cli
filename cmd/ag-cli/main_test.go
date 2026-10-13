package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDoctorBypassesCredentialInitialization(t *testing.T) {
	if os.Getenv("AG_TEST_DOCTOR_HELPER") == "1" {
		os.Args = []string{"ag-cli", "--raw-output", os.Getenv("AG_TEST_DOCTOR_COMMAND"), "--json"}
		main()
		return
	}
	for _, command := range []string{"doctor", "health"} {
		for _, body := range []string{`{"user":"alice","access_token":"synthetic-private-token"}`, `{"broken":"synthetic-private-token",`} {
			t.Run(command+"/"+body[:8], func(t *testing.T) {
				home := t.TempDir()
				dir := filepath.Join(home, "ag-cli")
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"aliases":{"health":"doctor"}}`), 0o600); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, "token.json")
				if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(os.Args[0], "-test.run=^TestDoctorBypassesCredentialInitialization$")
				cmd.Env = append(os.Environ(), "AG_TEST_DOCTOR_HELPER=1", "AG_TEST_DOCTOR_COMMAND="+command, "HOME="+home, "USERPROFILE="+home, "XDG_CONFIG_HOME="+home)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				err := cmd.Run()
				if runtime.GOOS != "windows" && err == nil {
					t.Fatal("expected failed permissions/config check")
				}
				var report struct {
					Checks []struct {
						ID string `json:"id"`
					} `json:"checks"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || len(report.Checks) == 0 {
					t.Fatalf("doctor did not run: %s %s", stdout.String(), stderr.String())
				}
				if strings.Contains(stdout.String()+stderr.String(), "synthetic-private-token") {
					t.Fatal("secret leaked")
				}
				got, _ := os.ReadFile(path)
				if string(got) != body {
					t.Fatal("credentials migrated or modified")
				}
				info, _ := os.Stat(path)
				if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
					t.Fatal("permissions repaired by startup")
				}
			})
		}
	}
}

func TestMainDisplaysHelp(t *testing.T) {
	if os.Getenv("AG_TEST_MAIN_HELPER") == "1" {
		os.Args = []string{"ag-cli", "--help"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainDisplaysHelp")
	cmd.Env = append(os.Environ(),
		"AG_TEST_MAIN_HELPER=1",
		"HOME="+t.TempDir(),
		"XDG_CONFIG_HOME=",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("main helper failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Work seamlessly with AtomGit") {
		t.Fatalf("help output did not contain command description:\n%s", output)
	}
}
