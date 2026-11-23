package main

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusQueued     TaskStatus = "queued" // Accepted for processing and waiting for a worker.
	StatusProcessing TaskStatus = "processing"
	StatusCompleted  TaskStatus = "completed"
)

type AnalysisResult struct {
	WordCount      int
	CharacterCount int
}

type Task struct {
	ID        int
	Input     string
	Status    TaskStatus
	Result    *AnalysisResult
	LastError string // Keeps the last processing error until the next submission.
}
