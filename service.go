package main

import (
	"errors"
	"sync"
)

const (
	workerCount   = 2
	queueCapacity = 4 // Counts waiting jobs; each worker can also hold one active job.
)

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrInvalidTaskState = errors.New("task must be pending or failed")
	ErrQueueFull        = errors.New("task queue is full")
	ErrStopped          = errors.New("task service has stopped")
)

// Processor implementations may be called by both workers at the same time.
type Processor interface {
	Process(text string) (AnalysisResult, error)
}

type job struct {
	id   int
	done chan error // Set only for callers of the synchronous Process method.
}

type TaskService struct {
	mu        sync.RWMutex // Protects task records, ID allocation, and the shutdown flag.
	tasks     map[int]Task
	nextID    int
	processor Processor
	queue     chan job
	space     *sync.Cond
	stopped   bool
	workers   sync.WaitGroup
}

func NewTaskService(processor Processor) *TaskService {
	s := &TaskService{
		tasks:     make(map[int]Task),
		nextID:    1,
		processor: processor,
		queue:     make(chan job, queueCapacity),
	}
	s.space = sync.NewCond(&s.mu)
	// Start the workers once. Each serves the same queue until it is drained.
	for range workerCount {
		s.workers.Go(s.work)
	}
	return s
}

func (s *TaskService) Create(text string) (Task, error) {
	if err := validateText(text); err != nil {
		return Task{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return Task{}, ErrStopped
	}
	task := Task{ID: s.nextID, Input: text, Status: StatusPending}
	s.tasks[task.ID] = task
	s.nextID++
	return task, nil
}

func (s *TaskService) Get(id int) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
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

// Submit accepts work immediately or returns ErrQueueFull without changing the task.
func (s *TaskService) Submit(id int) error {
	return s.enqueue(job{id: id}, false)
}

// SubmitWait lets batch callers wait for space while workers process earlier inputs.
func (s *TaskService) SubmitWait(id int) error {
	return s.enqueue(job{id: id}, true)
}

// Hold the same lock through the state check and send so only one caller can queue a task.
func (s *TaskService) enqueue(j job, waitForSpace bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if s.stopped {
			return ErrStopped
		}
		task, ok := s.tasks[j.id]
		if !ok {
			return ErrTaskNotFound
		}
		if task.Status != StatusPending && task.Status != StatusFailed {
			return ErrInvalidTaskState
		}
		select {
		case s.queue <- j:
			task.Status = StatusQueued
			task.LastError = ""
			s.tasks[j.id] = task
			return nil
		default:
			if !waitForSpace {
				return ErrQueueFull
			}
			// Wait releases mu. After waking, recheck: another sender may have taken the slot.
			s.space.Wait()
		}
	}
}

// Process preserves the synchronous Checkpoint 1 API using the same pool as Submit.
func (s *TaskService) Process(id int) (Task, error) {
	done := make(chan error, 1)
	if err := s.enqueue(job{id: id, done: done}, true); err != nil {
		return Task{}, err
	}
	if err := <-done; err != nil {
		return Task{}, err
	}
	return s.Get(id)
}

func (s *TaskService) work() {
	for j := range s.queue {
		s.mu.Lock()
		s.space.Broadcast()
		task := s.tasks[j.id]
		task.Status = StatusProcessing
		s.tasks[j.id] = task
		s.mu.Unlock()

		// Analyze outside the lock so the other worker and readers can keep running.
		result, err := s.processor.Process(task.Input)
		if err != nil {
			task.Status = StatusFailed
			task.LastError = err.Error()
		} else {
			task.Status = StatusCompleted
			task.Result = &result
		}

		s.mu.Lock()
		s.tasks[j.id] = task
		s.mu.Unlock()
		if j.done != nil {
			j.done <- err
		}
	}
}

func (s *TaskService) Shutdown() {
	s.mu.Lock()
	if !s.stopped {
		s.stopped = true
		// Workers can still receive jobs already in the buffer after close.
		close(s.queue)
		s.space.Broadcast()
	}
	s.mu.Unlock()
	// Workers need mu to store their final results, so wait after unlocking.
	s.workers.Wait()
}
