package engine

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/policy"
)

type DiscoveryEngine struct {
	policy *policy.Policy
}

func NewDiscoveryEngine(p *policy.Policy) *DiscoveryEngine {
	return &DiscoveryEngine{policy: p}
}

// Discover finds all leaf folders containing at least MinCFiles .c files.
func (e *DiscoveryEngine) Discover(root string) ([]domain.Submission, error) {
	var submissions []domain.Submission

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if strings.HasPrefix(d.Name(), ".") && path != absRoot {
			return filepath.SkipDir
		}

		isLeaf, cFiles, err := e.checkLeafFolder(path)
		if err != nil {
			return err
		}

		if isLeaf && len(cFiles) >= e.policy.Discover.MinCFiles {
			relPath, err := filepath.Rel(absRoot, path)
			if err != nil {
				relPath = d.Name()
			}

			submissions = append(submissions, domain.NewSubmission(
				relPath,
				path,
				cFiles,
				e.checkMissingFiles(cFiles),
			))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return submissions, nil
}

func (e *DiscoveryEngine) checkMissingFiles(cFiles []string) []string {
	if len(e.policy.RequiredFiles) == 0 {
		return nil
	}

	// Case-insensitive lookup
	present := make(map[string]bool)
	for _, f := range cFiles {
		present[strings.ToLower(f)] = true
	}

	var missing []string
	for _, req := range e.policy.RequiredFiles {
		if !present[strings.ToLower(req)] {
			missing = append(missing, req)
		}
	}
	return missing
}

// checkLeafFolder returns true if dir has no non-hidden subdirectories.
func (e *DiscoveryEngine) checkLeafFolder(dir string) (bool, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, nil, err
	}

	var cFiles []string
	hasSubdirs := false

	for _, entry := range entries {
		if entry.IsDir() {
			if !strings.HasPrefix(entry.Name(), ".") {
				hasSubdirs = true
			}
			continue
		}

		if strings.HasSuffix(strings.ToLower(entry.Name()), ".c") {
			cFiles = append(cFiles, entry.Name())
		}
	}

	return !hasSubdirs, cFiles, nil
}

