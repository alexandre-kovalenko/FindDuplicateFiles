package Helper

import (
	"fmt"
	"path/filepath"
)

func ValidateAbsolutePath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing to delete relative path: %s", path)
	}
	return nil
}
