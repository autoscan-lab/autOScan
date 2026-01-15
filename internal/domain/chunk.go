package domain

// Chunk represents a slice of text from a file with its embedding
type Chunk struct {
	ID        string
	CorpusID  string
	FilePath  string
	Content   string
	StartLine int
	EndLine   int
	Embedding []float32
	Metadata  ChunkMetadata
}

// ChunkMetadata holds additional information about a chunk
type ChunkMetadata struct {
	FileType  string
	CreatedAt int64
	UpdatedAt int64
}
