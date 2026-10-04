package config

import (
	"errors"
	"fmt"
	"os"
)

// removeTemporaryConfig cleans up a file still owned by an unfinished save.
// An empty path means the file has already been renamed to its destination.
func removeTemporaryConfig(path string, resultErr *error) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		*resultErr = errors.Join(*resultErr, fmt.Errorf("remove temporary config file: %w", err))
	}
}
