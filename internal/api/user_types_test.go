package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUpdateUserProfileRequestJSON(t *testing.T) {
	empty := ""
	nickname := "Alice"
	request := UpdateUserProfileRequest{
		Nickname:    &nickname,
		Description: &empty,
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"nickname": "Alice", "description": ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON = %#v, want %#v", got, want)
	}
}

func TestUpdatedUserProfileJSONTags(t *testing.T) {
	profile := UpdatedUserProfile{GitHubAccount: "alice-gh", Login: "alice"}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["github_account"] != "alice-gh" || got["login"] != "alice" {
		t.Fatalf("JSON = %#v", got)
	}
}
