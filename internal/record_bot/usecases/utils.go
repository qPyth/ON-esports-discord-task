package usecases

import (
	"errors"
	"fmt"
	"os"
)

var errDirNotEmpty = errors.New("dir not empty")

func removeEmptyDir(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) > 0 {
		return errDirNotEmpty
	}

	err = os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("failed to remove directory: %w", err)
	}
	return nil
}
