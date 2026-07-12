package auth

import (
	"errors"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type testConfig struct{}

func (testConfig) GetToken() (string, error) { return "", errors.New("not authenticated") }
func (testConfig) GetUser() (string, error)  { return "", errors.New("not authenticated") }
func (testConfig) GetHost() string           { return "atomgit.com" }

func TestNewCmdAuthRegistersSubcommands(t *testing.T) {
	cmd := NewCmdAuth(&cmdutil.Factory{Config: testConfig{}})
	want := map[string]bool{"login": false, "logout": false, "refresh": false, "status": false, "token": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q was not registered", name)
		}
	}
	login, _, err := cmd.Find([]string{"login"})
	if err != nil {
		t.Fatal(err)
	}
	if login.Flags().Lookup("force") == nil {
		t.Fatal("login --force flag was not registered")
	}
}
