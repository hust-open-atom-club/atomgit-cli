package license

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestNewCmdLicense(t *testing.T) {
	cmd := NewCmdLicense(&cmdutil.Factory{})
	check, _, err := cmd.Find([]string{"check"})
	if err != nil {
		t.Fatal(err)
	}
	if check.Name() != "check" {
		t.Fatalf("command = %q", check.Name())
	}
	if err := check.Args(check, nil); err == nil {
		t.Fatal("check accepted no license")
	}
	if err := check.Args(check, []string{"MIT"}); err != nil {
		t.Fatalf("check rejected one license: %v", err)
	}
	if err := check.Args(check, []string{"MIT", "Apache-2.0"}); err == nil {
		t.Fatal("check accepted too many licenses")
	}
}
