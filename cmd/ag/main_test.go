package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainDisplaysHelp(t *testing.T) {
	if os.Getenv("AG_TEST_MAIN_HELPER") == "1" {
		os.Args = []string{"ag", "--help"}
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
