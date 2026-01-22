// Package domain contains the core business types for autOScan.
package domain

// Submission represents a student's lab submission folder.
type Submission struct {
	// ID is the relative path from root (e.g., "student1" or "labA/student1")
	ID string

	// Path is the absolute path to the submission folder
	Path string

	// CFiles is the list of .c files found in this submission
	CFiles []string

	// MissingFiles is the list of required files that are missing
	MissingFiles []string
}

// NewSubmission creates a new Submission.
func NewSubmission(id, path string, cFiles, missingFiles []string) Submission {
	return Submission{
		ID:           id,
		Path:         path,
		CFiles:       cFiles,
		MissingFiles: missingFiles,
	}
}

// HasMissingFiles returns true if required files are missing.
func (s Submission) HasMissingFiles() bool {
	return len(s.MissingFiles) > 0
}
