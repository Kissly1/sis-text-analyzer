package main

import "errors"

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrInvalidTaskState = errors.New("task must be pending")
)

type Processor interface {
	Process(text string) (AnalysisResult, error)
}

type TaskService struct {
	tasks     map[int]Task
	nextID    int
	processor Processor
}

func NewTaskService(processor Processor) *TaskService {
	return &TaskService{
		tasks:     make(map[int]Task),
		nextID:    1,
		processor: processor,
	}
}

func (s *TaskService) Create(text string) (Task, error) {
	if err := validateText(text); err != nil {
		return Task{}, err
	}

	task := Task{
		ID:     s.nextID,
		Input:  text,
		Status: StatusPending,
	}

	s.tasks[task.ID] = task
	s.nextID++

	return task, nil
}

func (s *TaskService) Get(id int) (Task, error) {
	task, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrTaskNotFound
	}

	if task.Result != nil {
		resultCopy := *task.Result
		task.Result = &resultCopy
	}

	return task, nil
}

func (s *TaskService) Process(id int) (Task, error) {
	task, err := s.Get(id)
	if err != nil {
		return Task{}, err
	}

	if task.Status != StatusPending {
		return Task{}, ErrInvalidTaskState
	}

	task.Status = StatusProcessing
	s.tasks[id] = task

	result, err := s.processor.Process(task.Input)
	if err != nil {
		task.Status = StatusPending
		s.tasks[id] = task
		return Task{}, err
	}

	task.Result = &result
	task.Status = StatusCompleted
	s.tasks[id] = task

	return s.Get(id)
}
