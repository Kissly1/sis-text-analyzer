package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

type processorFunc func(string) (AnalysisResult, error)

func (f processorFunc) Process(text string) (AnalysisResult, error) { return f(text) }

func createTask(t *testing.T, s *TaskService, text string) Task {
	t.Helper()
	task, err := s.Create(text)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func checkState(t *testing.T, s *TaskService, id int, want TaskStatus) Task {
	t.Helper()
	task, err := s.Get(id)
	if err != nil || task.Status != want {
		t.Fatalf("Get(%d) = %+v, %v; want %s", id, task, err, want)
	}
	if want != StatusCompleted && task.Result != nil {
		t.Fatalf("unfinished task %d has a result", id)
	}
	return task
}

// Hold active processors on a channel so queue occupancy and overlap are observable.
func TestPoolCapacityAndShutdown(t *testing.T) {
	for _, stopWhenFull := range []bool{false, true} {
		t.Run(fmt.Sprintf("stop_when_full_%t", stopWhenFull), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				finish := sync.OnceFunc(func() { close(release) })
				s := NewTaskService(processorFunc(func(text string) (AnalysisResult, error) {
					<-release
					return TextAnalyzer{}.Process(text)
				}))
				defer s.Shutdown()
				defer finish()
				if err := s.Submit(100); !errors.Is(err, ErrTaskNotFound) {
					t.Fatalf("unknown task: %v", err)
				}
				var accepted []Task
				for i := range workerCount + queueCapacity {
					task := createTask(t, s, fmt.Sprintf("текст %0*d", i+1, i))
					if err := s.Submit(task.ID); err != nil {
						t.Fatal(err)
					}
					synctest.Wait()
					want := StatusProcessing
					if i >= workerCount {
						want = StatusQueued
					}
					checkState(t, s, task.ID, want)
					if err := s.Submit(task.ID); !errors.Is(err, ErrInvalidTaskState) {
						t.Fatalf("duplicate %s task: %v", want, err)
					}
					accepted = append(accepted, task)
				}

				// A rejected job stays pending and can be submitted again.
				extra := createTask(t, s, "ещё один текст")
				if err := s.Submit(extra.ID); !errors.Is(err, ErrQueueFull) {
					t.Fatalf("full queue: %v", err)
				}
				checkState(t, s, extra.ID, StatusPending)
				submitted := make(chan error, 1)
				go func() { submitted <- s.SubmitWait(extra.ID) }()
				synctest.Wait()
				select {
				case err := <-submitted:
					t.Fatalf("full queue did not block submission: %v", err)
				default:
				}
				checkState(t, s, extra.ID, StatusPending)
				if stopWhenFull {
					stopped := make(chan struct{})
					go func() { s.Shutdown(); close(stopped) }()
					synctest.Wait()
					if err := <-submitted; !errors.Is(err, ErrStopped) {
						t.Fatalf("blocked submit during shutdown: %v", err)
					}
					select {
					case <-stopped:
						t.Fatal("shutdown returned with unfinished work")
					default:
					}
					finish()
					<-stopped
					checkState(t, s, extra.ID, StatusPending)
				} else {
					finish()
					if err := <-submitted; err != nil {
						t.Fatal(err)
					}
					accepted = append(accepted, extra)
					synctest.Wait()
					if err := s.Submit(extra.ID); !errors.Is(err, ErrInvalidTaskState) {
						t.Fatalf("completed task submitted again: %v", err)
					}
				}

				s.Shutdown()
				for _, original := range accepted {
					task := checkState(t, s, original.ID, StatusCompleted)
					want, _ := TextAnalyzer{}.Process(original.Input)
					if task.Result == nil || *task.Result != want {
						t.Fatalf("task %d: got %v, want %v", task.ID, task.Result, want)
					}
				}
				if err := s.Submit(extra.ID); !errors.Is(err, ErrStopped) {
					t.Fatalf("submit after shutdown: %v", err)
				}
			})
		})
	}
}

func TestConcurrentClients(t *testing.T) {
	s := NewTaskService(TextAnalyzer{})
	defer s.Shutdown()
	const clients = 48
	ids := make(chan int, clients)
	var group sync.WaitGroup
	for i := range clients {
		group.Go(func() {
			text := fmt.Sprintf("word %d", i)
			task, err := s.Create(text)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- task.ID
			if err := s.SubmitWait(task.ID); err != nil {
				t.Error(err)
				return
			}
			snapshot, err := s.Get(task.ID)
			if err != nil || snapshot.Input != text {
				t.Errorf("concurrent read: %+v, %v", snapshot, err)
			}
		})
	}
	group.Wait()
	s.Shutdown()
	close(ids)
	seen := make(map[int]bool)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate ID %d", id)
		}
		seen[id] = true
		task := checkState(t, s, id, StatusCompleted)
		want, _ := TextAnalyzer{}.Process(task.Input)
		if task.Result == nil || *task.Result != want {
			t.Fatalf("task %d: unexpected result %v", id, task.Result)
		}
		// Changing a snapshot must leave the stored result intact.
		task.Result.WordCount = -1
		again, _ := s.Get(id)
		if *again.Result != want {
			t.Fatalf("Get(%d) exposed the stored result", id)
		}
	}
	if len(seen) != clients {
		t.Fatalf("got %d unique tasks", len(seen))
	}
}

func TestConcurrentDuplicateSubmit(t *testing.T) {
	var calls atomic.Int32
	s := NewTaskService(processorFunc(func(text string) (AnalysisResult, error) {
		calls.Add(1)
		return TextAnalyzer{}.Process(text)
	}))
	defer s.Shutdown()
	task := createTask(t, s, "once")
	var accepted atomic.Int32
	var group sync.WaitGroup
	start := make(chan struct{})
	for range 32 {
		group.Go(func() {
			<-start
			switch err := s.Submit(task.ID); {
			case err == nil:
				accepted.Add(1)
			case !errors.Is(err, ErrInvalidTaskState):
				t.Errorf("unexpected submission error: %v", err)
			}
		})
	}
	// Release all callers together to compete for the same pending task.
	close(start)
	group.Wait()
	s.Shutdown()
	if accepted.Load() != 1 || calls.Load() != 1 {
		t.Fatalf("accepted %d, processed %d", accepted.Load(), calls.Load())
	}
}

func TestAnalysisFailureCanBeRetried(t *testing.T) {
	failure := errors.New("analysis unavailable")
	var calls atomic.Int32
	s := NewTaskService(processorFunc(func(text string) (AnalysisResult, error) {
		if calls.Add(1) == 1 {
			return AnalysisResult{}, failure
		}
		return TextAnalyzer{}.Process(text)
	}))
	defer s.Shutdown()
	task := createTask(t, s, "try again")
	if _, err := s.Process(task.ID); !errors.Is(err, failure) {
		t.Fatalf("first attempt returned %v", err)
	}
	if got := checkState(t, s, task.ID, StatusFailed); got.LastError != failure.Error() {
		t.Fatalf("failure was not recorded: %+v", got)
	}
	if err := s.Submit(task.ID); err != nil {
		t.Fatal(err)
	}
	s.Shutdown()
	got := checkState(t, s, task.ID, StatusCompleted)
	if got.LastError != "" || got.Result == nil || *got.Result != (AnalysisResult{2, 9}) {
		t.Fatalf("unexpected retry result: %+v", got)
	}
}

func TestSubmitRacesWithShutdown(t *testing.T) {
	s := NewTaskService(TextAnalyzer{})
	defer s.Shutdown()
	type attempt struct {
		id  int
		err error
	}
	const count = 32
	results := make(chan attempt, count)
	start := make(chan struct{})
	var group sync.WaitGroup
	for i := range count {
		task := createTask(t, s, fmt.Sprintf("task %d", i))
		group.Go(func() {
			<-start
			results <- attempt{task.ID, s.Submit(task.ID)}
		})
	}
	for range 2 {
		group.Go(func() {
			<-start
			s.Shutdown()
		})
	}
	close(start)
	group.Wait()
	close(results)
	for result := range results {
		switch {
		case result.err == nil:
			checkState(t, s, result.id, StatusCompleted)
		case errors.Is(result.err, ErrStopped), errors.Is(result.err, ErrQueueFull):
			checkState(t, s, result.id, StatusPending)
		default:
			t.Fatalf("task %d: %v", result.id, result.err)
		}
	}
}
