// Package engine contains the core processing logic for autOScan.
package engine

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/policy"
)

// DiscoveryEngine finds student submissions in a folder structure.
type DiscoveryEngine struct {
	policy *policy.Policy
}

// NewDiscoveryEngine creates a new discovery engine.
func NewDiscoveryEngine(p *policy.Policy) *DiscoveryEngine {
	return &DiscoveryEngine{policy: p}
}

// Discover finds all submissions under the given root directory.
// A submission is a leaf folder containing at least MinCFiles .c files.
func (e *DiscoveryEngine) Discover(root string) ([]domain.Submission, error) {
	var submissions []domain.Submission

	// Get absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip files, we only care about directories
		if !d.IsDir() {
			return nil
		}

		// Skip hidden directories
		if strings.HasPrefix(d.Name(), ".") && path != absRoot {
			return filepath.SkipDir
		}

		// Check if this is a leaf submission folder
		isLeaf, cFiles, err := e.checkLeafFolder(path)
		if err != nil {
			return err
		}

		if isLeaf && len(cFiles) >= e.policy.Discover.MinCFiles {
			// Calculate relative path for ID
			relPath, err := filepath.Rel(absRoot, path)
			if err != nil {
				relPath = d.Name()
			}

			submissions = append(submissions, domain.NewSubmission(
				relPath,
				path,
				cFiles,
			))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return submissions, nil
}

// checkLeafFolder determines if a directory is a leaf folder (no subdirectories)
// and returns the list of .c files it contains.
func (e *DiscoveryEngine) checkLeafFolder(dir string) (bool, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, nil, err
	}

	var cFiles []string
	hasSubdirs := false

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories when checking for subdirs
			if !strings.HasPrefix(entry.Name(), ".") {
				hasSubdirs = true
			}
			continue
		}

		// Check for .c files
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".c") {
			cFiles = append(cFiles, entry.Name())
		}
	}

	// It's a leaf if it has no subdirectories (or only hidden ones)
	isLeaf := !hasSubdirs

	return isLeaf, cFiles, nil
}

// DiscoverQuick returns just the count of submissions without full details.
// Useful for progress display.
func (e *DiscoveryEngine) DiscoverQuick(root string) (int, error) {
	subs, err := e.Discover(root)
	if err != nil {
		return 0, err
	}
	return len(subs), nil
}
