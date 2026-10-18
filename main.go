package main

import (
	"fmt"
	"os"
)

func main() {
	texts := os.Args[1:]
	if len(texts) == 0 {
		texts = []string{
			"Hello Go world",
			"Привет мир",
		}
	}

	service := NewTaskService(TextAnalyzer{})

	for _, text := range texts {
		task, err := service.Create(text)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Create error:", err)
			os.Exit(1)
		}

		fmt.Printf("Created task %d: %s\n", task.ID, task.Status)

		task, err = service.Process(task.ID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Processing error:", err)
			os.Exit(1)
		}

		fmt.Printf(
			"Task %d: %s | words=%d | characters=%d\n",
			task.ID,
			task.Status,
			task.Result.WordCount,
			task.Result.CharacterCount,
		)
	}
}
