package fsutil

import (
	"os"
	"path/filepath"
)

// WalkFiles walks a directory and returns all files matching the filter
func WalkFiles(root string, filter func(path string) bool) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filter(path) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// IsTextFile checks if a file is likely a text file
func IsTextFile(path string) bool {
	ext := filepath.Ext(path)
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true,
		".js": true, ".ts": true, ".json": true, ".yaml": true,
		".yml": true, ".toml": true, ".html": true, ".css": true,
		".rs": true, ".java": true, ".c": true, ".cpp": true,
		".h": true, ".rb": true, ".php": true, ".sh": true,
	}
	return textExts[ext]
}
