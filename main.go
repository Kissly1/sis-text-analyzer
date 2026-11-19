package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(texts []string) error {
	if len(texts) == 0 {
		texts = []string{"Hello Go world", "Привет мир"}
	}
	service := NewTaskService(TextAnalyzer{})
	defer service.Shutdown()

	ids := make([]int, 0, len(texts))
	for _, text := range texts {
		task, err := service.Create(text)
		if err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		if err := service.Submit(task.ID); err != nil {
			return fmt.Errorf("submit task %d: %w", task.ID, err)
		}
		ids = append(ids, task.ID)
	}

	service.Shutdown()
	for _, id := range ids {
		task, err := service.Get(id)
		if err != nil {
			return err
		}
		if task.Status != StatusCompleted {
			return fmt.Errorf("task %d: %s", id, task.LastError)
		}
		fmt.Printf("Task %d: %s | words=%d | characters=%d\n",
			id, task.Status, task.Result.WordCount, task.Result.CharacterCount)
	}
	return nil
}
