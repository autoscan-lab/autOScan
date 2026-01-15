package domain

// Document represents a file in a corpus
type Document struct {
	ID       string
	CorpusID string
	Path     string
	Name     string
	Type     string
	Size     int64
	Hash     string
}
