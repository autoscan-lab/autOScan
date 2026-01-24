package domain

type Submission struct {
	ID           string
	Path         string
	CFiles       []string
	MissingFiles []string
}

func NewSubmission(id, path string, cFiles, missingFiles []string) Submission {
	return Submission{ID: id, Path: path, CFiles: cFiles, MissingFiles: missingFiles}
}

func (s Submission) HasMissingFiles() bool { return len(s.MissingFiles) > 0 }
