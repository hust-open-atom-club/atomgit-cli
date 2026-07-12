package repo

import "testing"

func TestParseRepoArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantURL  string
		wantName string
	}{
		{
			name:     "HTTPS URL",
			arg:      "https://atomgit.com/owner/project.git",
			wantURL:  "https://atomgit.com/owner/project.git",
			wantName: "project",
		},
		{
			name:     "HTTP URL",
			arg:      "http://atomgit.com/owner/project",
			wantURL:  "http://atomgit.com/owner/project",
			wantName: "project",
		},
		{
			name:     "SSH URL",
			arg:      "git@atomgit.com:owner/project.git",
			wantURL:  "git@atomgit.com:owner/project.git",
			wantName: "project",
		},
		{
			name:     "owner and repository",
			arg:      "owner/project",
			wantURL:  "https://atomgit.com/owner/project.git",
			wantName: "project",
		},
		{
			name:     "repository only",
			arg:      "project",
			wantURL:  "https://atomgit.com/project",
			wantName: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotName := parseRepoArg(tt.arg)
			if gotURL != tt.wantURL || gotName != tt.wantName {
				t.Fatalf("parseRepoArg(%q) = (%q, %q), want (%q, %q)", tt.arg, gotURL, gotName, tt.wantURL, tt.wantName)
			}
		})
	}
}
