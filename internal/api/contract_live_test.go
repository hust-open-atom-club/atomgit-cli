//go:build contractlive

package api_test

import (
	"os"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/apicontract"
)

// Both a build tag and an explicit opt-in are required; default go test ./...
// cannot access the network even if credential variables happen to be set.
func TestLiveAPIContracts(t *testing.T) {
	if os.Getenv("ATOMGIT_CONTRACT_LIVE") != "1" {
		t.Skip("set ATOMGIT_CONTRACT_LIVE=1 for dedicated-account read-only checks")
	}
	token := os.Getenv("ATOMGIT_CONTRACT_TOKEN")
	parts := strings.Split(os.Getenv("ATOMGIT_CONTRACT_REPO"), "/")
	if len(parts) != 2 || token == "" {
		t.Fatal("ATOMGIT_CONTRACT_REPO=owner/repo and ATOMGIT_CONTRACT_TOKEN are required")
	}
	for _, f := range contractFixtures(t) {
		switch f.Name {
		case "repository", "issue-list", "workflows":
		default:
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			if err := apicontract.Probe(t.Context(), nil, f, parts[0], parts[1], token); err != nil {
				t.Fatal(err)
			}
		})
	}
}
