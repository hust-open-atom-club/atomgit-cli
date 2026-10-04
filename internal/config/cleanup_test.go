package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveTemporaryConfig(t *testing.T) {
	for _, kind := range []string{"file", "missing", "renamed", "nonempty directory"} {
		for _, failed := range []bool{false, true} {
			t.Run(kind+"/"+map[bool]string{false: "success", true: "failed save"}[failed], func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "temporary")
				switch kind {
				case "file", "renamed":
					if err := os.WriteFile(path, []byte("config"), 0o600); err != nil {
						t.Fatal(err)
					}
				case "nonempty directory":
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, "keep"), nil, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				var original error
				if failed {
					original = errors.New("save failed")
				}
				result := original
				cleanupPath := path
				if kind == "renamed" {
					cleanupPath = ""
				}
				removeTemporaryConfig(cleanupPath, &result)
				if kind == "nonempty directory" {
					var pathErr *os.PathError
					if !errors.As(result, &pathErr) || pathErr.Op != "remove" || !strings.Contains(result.Error(), "remove temporary config file") {
						t.Fatalf("cleanup error = %v", result)
					}
					if failed && !errors.Is(result, original) {
						t.Fatal("save error lost")
					}
					if !errors.Is(result, pathErr.Err) {
						t.Fatal("cleanup cause lost")
					}
				} else if !errors.Is(result, original) {
					t.Fatalf("successful cleanup changed error: %v", result)
				}
				_, err := os.Stat(path)
				if kind == "renamed" || kind == "nonempty directory" {
					if err != nil {
						t.Fatalf("owned destination was removed: %v", err)
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("temporary file remains: %v", err)
				}
			})
		}
	}
}

func TestReleaseAliasLock(t *testing.T) {
	saveErr := errors.New("save failed")
	unlockErr := errors.New("unlock failed")
	for _, tt := range []struct {
		name       string
		saved      bool
		original   error
		releaseErr error
	}{
		{"saved", true, nil, nil},
		{"unchanged", false, nil, nil},
		{"save failed", false, saveErr, nil},
		{"saved but unlock failed", true, nil, unlockErr},
		{"unchanged but unlock failed", false, nil, unlockErr},
		{"save and unlock failed", false, saveErr, unlockErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.original
			calls := 0
			releaseAliasLock(func() error { calls++; return tt.releaseErr }, tt.saved, &result)
			if calls != 1 {
				t.Fatalf("release calls = %d", calls)
			}
			if tt.releaseErr == nil {
				if !errors.Is(result, tt.original) {
					t.Fatalf("successful release changed error: %v", result)
				}
				return
			}
			if !errors.Is(result, unlockErr) || (tt.original != nil && !errors.Is(result, tt.original)) {
				t.Fatalf("missing cause in %v", result)
			}
			if strings.Contains(result.Error(), "was saved") != tt.saved {
				t.Fatalf("incorrect saved state in %v", result)
			}
		})
	}
}
