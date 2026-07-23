package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReleaseAuthorJSON(t *testing.T) {
	raw := `{
		"id": "6403465",
		"login": "mudongliang",
		"name": "mudongliang",
		"avatar_url": "https://cdn-img.gitcode.com/avatar.png",
		"html_url": "https://atomgit.com/mudongliang",
		"type": "User",
		"url": "https://api.atomgit.com/api/v5/users/mudongliang"
	}`

	var author ReleaseAuthor
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&author); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if author.ID != "6403465" {
		t.Fatalf("id = %q", author.ID)
	}
	if author.Type != "User" {
		t.Fatalf("type = %q", author.Type)
	}
	if author.URL != "https://api.atomgit.com/api/v5/users/mudongliang" {
		t.Fatalf("url = %q", author.URL)
	}
}

func TestReleaseAssetJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ReleaseAsset
	}{
		{
			name: "uploaded attachment is deletable",
			raw:  `{"id": 42, "name": "ag.tar.gz", "type": "attach", "browser_download_url": "https://raw.atomgit.com/o/ag.tar.gz"}`,
			want: ReleaseAsset{ID: 42, Name: "ag.tar.gz", Type: "attach", BrowserDownloadURL: "https://raw.atomgit.com/o/ag.tar.gz"},
		},
		{
			name: "source archive has id 0 and is not deletable",
			raw:  `{"id": 0, "name": "v0.2.zip", "type": "source", "browser_download_url": "https://raw.atomgit.com/o/v0.2.zip"}`,
			want: ReleaseAsset{ID: 0, Name: "v0.2.zip", Type: "source", BrowserDownloadURL: "https://raw.atomgit.com/o/v0.2.zip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ReleaseAsset
			if err := json.Unmarshal([]byte(tt.raw), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got != tt.want {
				t.Fatalf("asset = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestReleaseJSON(t *testing.T) {
	raw := `{
		"tag_name": "v0.2",
		"target_commitish": "b635b9a8a2c27bd86a93801bb6835dfa5744288d",
		"prerelease": false,
		"name": "v0.2",
		"body": "## 更新内容\n",
		"release_status": "latest",
		"created_at": "2026-07-11T00:25:46+08:00",
		"author": {"login": "mudongliang", "type": "User"},
		"assets": [
			{"id": 1, "name": "v0.2.zip", "type": "source", "browser_download_url": "https://raw.atomgit.com/o/v0.2.zip"},
			{"id": 7, "name": "ag-linux-amd64.tar.gz", "type": "attach", "browser_download_url": "https://raw.atomgit.com/o/ag-linux-amd64.tar.gz"}
		]
	}`

	var got Release
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.TagName != "v0.2" || got.Name != "v0.2" {
		t.Fatalf("identity = %#v", got)
	}
	if got.Prerelease {
		t.Fatal("prerelease = true, want false")
	}
	if got.TargetCommitish != "b635b9a8a2c27bd86a93801bb6835dfa5744288d" {
		t.Fatalf("target_commitish = %q", got.TargetCommitish)
	}
	if got.ReleaseStatus != "latest" {
		t.Fatalf("release_status = %q, want latest", got.ReleaseStatus)
	}
	if got.Author.Login != "mudongliang" || got.Author.Type != "User" {
		t.Fatalf("author = %#v", got.Author)
	}
	if len(got.Assets) != 2 {
		t.Fatalf("assets len = %d", len(got.Assets))
	}
	if got.Assets[1].ID != 7 || got.Assets[1].Type != "attach" {
		t.Fatalf("attach asset = %#v", got.Assets[1])
	}
}

func TestCreateReleaseRequestJSON(t *testing.T) {
	tests := []struct {
		name string
		req  CreateReleaseRequest
		want string
	}{
		{
			name: "minimal required fields keep empty body",
			req:  CreateReleaseRequest{TagName: "v1.0.0", Name: "v1.0.0", Body: ""},
			want: `{"tag_name":"v1.0.0","name":"v1.0.0","body":""}`,
		},
		{
			name: "target and prerelease status included when set",
			req:  CreateReleaseRequest{TagName: "v1.0.0", Name: "v1.0.0", Body: "notes", TargetCommitish: "main", ReleaseStatus: ReleaseStatusPre},
			want: `{"tag_name":"v1.0.0","name":"v1.0.0","body":"notes","target_commitish":"main","release_status":"pre"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestUpdateReleaseRequestJSON(t *testing.T) {
	tests := []struct {
		name string
		req  UpdateReleaseRequest
		want string
	}{
		{
			name: "required name and body only",
			req:  UpdateReleaseRequest{Name: "v1.0.0", Body: "notes"},
			want: `{"name":"v1.0.0","body":"notes"}`,
		},
		{
			name: "release_status latest included when set",
			req:  UpdateReleaseRequest{Name: "v1.0.0", Body: "notes", ReleaseStatus: ReleaseStatusLatest},
			want: `{"name":"v1.0.0","body":"notes","release_status":"latest"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestReleaseUploadURLJSON(t *testing.T) {
	raw := `{"url": "https://store.example.com/upload", "headers": {"Content-Type": "application/octet-stream", "X-MD5": "abc"}}`

	var got ReleaseUploadURL
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.URL != "https://store.example.com/upload" {
		t.Fatalf("url = %q", got.URL)
	}
	if got.Headers["Content-Type"] != "application/octet-stream" {
		t.Fatalf("Content-Type header = %q", got.Headers["Content-Type"])
	}
	if got.Headers["X-MD5"] != "abc" {
		t.Fatalf("X-MD5 header = %q", got.Headers["X-MD5"])
	}
}
