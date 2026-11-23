package main

import (
	"errors"
	"testing"
)

func TestTaskServiceCreateAndGet(t *testing.T) {
	service := NewTaskService(TextAnalyzer{})
	t.Cleanup(service.Shutdown) // Each service now starts workers that the test must stop.
	text := "Hello Go"

	created, err := service.Create(text)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if created.ID != 1 || created.Input != text {
		t.Fatalf("unexpected created task: %+v", created)
	}

	if created.Status != StatusPending {
		t.Errorf("expected pending, got %s", created.Status)
	}

	if created.Result != nil {
		t.Error("new task must not have a result")
	}

	stored, err := service.Get(created.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}

	if stored != created {
		t.Errorf("expected %+v, got %+v", created, stored)
	}
}

func TestTaskServiceCreateInvalidInput(t *testing.T) {
	inputs := []string{"", "   ", "\t\n"}

	for _, input := range inputs {
		service := NewTaskService(TextAnalyzer{})
		t.Cleanup(service.Shutdown)

		_, err := service.Create(input)
		if !errors.Is(err, ErrEmptyText) {
			t.Errorf("input %q: expected ErrEmptyText, got %v", input, err)
		}

		_, err = service.Get(1)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("input %q: invalid task was stored", input)
		}
	}
}

func TestTaskServiceGetNotFound(t *testing.T) {
	service := NewTaskService(TextAnalyzer{})
	t.Cleanup(service.Shutdown)

	_, err := service.Get(999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskServiceProcess(t *testing.T) {
	service := NewTaskService(TextAnalyzer{})
	t.Cleanup(service.Shutdown)

	task, err := service.Create("Hello Go")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	processed, err := service.Process(task.ID)
	if err != nil {
		t.Fatalf("process task: %v", err)
	}

	want := AnalysisResult{
		WordCount:      2,
		CharacterCount: 8,
	}

	if processed.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", processed.Status)
	}

	if processed.Result == nil || *processed.Result != want {
		t.Fatalf("unexpected result: %+v", processed.Result)
	}

	stored, err := service.Get(task.ID)
	if err != nil {
		t.Fatalf("get processed task: %v", err)
	}

	if stored.Status != StatusCompleted {
		t.Errorf("stored status: expected completed, got %s", stored.Status)
	}

	if stored.Result == nil || *stored.Result != want {
		t.Errorf("result was not stored correctly: %+v", stored.Result)
	}

	_, err = service.Process(task.ID)
	if !errors.Is(err, ErrInvalidTaskState) {
		t.Errorf("expected ErrInvalidTaskState, got %v", err)
	}

	_, err = service.Process(999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskServiceIndependentTasks(t *testing.T) {
	service := NewTaskService(TextAnalyzer{})
	t.Cleanup(service.Shutdown)

	first, err := service.Create("Hello Go")
	if err != nil {
		t.Fatalf("create first task: %v", err)
	}

	second, err := service.Create("Go")
	if err != nil {
		t.Fatalf("create second task: %v", err)
	}

	if first.ID == second.ID {
		t.Fatal("tasks must have different IDs")
	}

	_, err = service.Process(first.ID)
	if err != nil {
		t.Fatalf("process first task: %v", err)
	}

	pending, err := service.Get(second.ID)
	if err != nil {
		t.Fatalf("get second task: %v", err)
	}

	if pending.Input != "Go" ||
		pending.Status != StatusPending ||
		pending.Result != nil {
		t.Fatalf("second task changed unexpectedly: %+v", pending)
	}

	processed, err := service.Process(second.ID)
	if err != nil {
		t.Fatalf("process second task: %v", err)
	}

	secondWant := AnalysisResult{WordCount: 1, CharacterCount: 2}

	if processed.Status != StatusCompleted ||
		processed.Result == nil ||
		*processed.Result != secondWant {
		t.Fatalf("unexpected second task: %+v", processed)
	}

	stored, err := service.Get(first.ID)
	if err != nil {
		t.Fatalf("get first task: %v", err)
	}

	firstWant := AnalysisResult{WordCount: 2, CharacterCount: 8}

	if stored.Input != "Hello Go" ||
		stored.Status != StatusCompleted ||
		stored.Result == nil ||
		*stored.Result != firstWant {
		t.Fatalf("first task changed unexpectedly: %+v", stored)
	}
}
