package main

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusQueued     TaskStatus = "queued"
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
	LastError string
}
