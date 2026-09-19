package api

import (
	"encoding/json"
	"testing"
)

func TestPullRequestReactionResponseShapes(t *testing.T) {
	for _, tt := range []struct{ name, body, id, content, emoji, emojiName string }{
		{"documented", `{"id":"6aadf74ba99efd72002e8dcd","emoji":"👀","emoji_name":"eyes"}`, "6aadf74ba99efd72002e8dcd", "eyes", "👀", "eyes"},
		{"numeric ID without precision loss", `{"id":9007199254740993,"content":"+1"}`, "9007199254740993", "+1", "", ""},
		{"numeric string preserves zeros", `{"id":"00123","emoji_name":"like"}`, "00123", "like", "", "like"},
		{"emoji only", `{"id":"a","emoji":"👍"}`, "a", "👍", "👍", ""},
		{"legacy content retained", `{"id":42,"content":"heart","emoji":"❤️","emoji_name":"love"}`, "42", "heart", "❤️", "love"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var r PullRequestReaction
			if err := json.Unmarshal([]byte(tt.body), &r); err != nil {
				t.Fatal(err)
			}
			if string(r.ID) != tt.id || r.GetContent() != tt.content || r.Emoji != tt.emoji || r.EmojiName != tt.emojiName {
				t.Fatalf("reaction = %#v, content = %q", r, r.GetContent())
			}
		})
	}
}

func TestPullRequestReactionRejectsInvalidRecords(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"id":null}`, `{"id":""}`, `{"id":" \t"}`, `{"id":true}`, `{"id":{}}`, `{"id":[]}`, `{"id":"a","emoji":42}`} {
		t.Run(body, func(t *testing.T) {
			var r PullRequestReaction
			if err := json.Unmarshal([]byte(body), &r); err == nil {
				t.Fatalf("accepted invalid record: %s", body)
			}
		})
	}
}
