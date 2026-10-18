# SIS Text Analyzer

An individual project for SIS Checkpoint 1 of the Golang Application Development course.

The application creates text-analysis tasks, stores them in memory, processes them synchronously, and saves their results.

## Requirements

- Go 1.27 or later.
- No external dependencies.

## Run

Run the commands from the project directory.

Start with two example texts:

```bash
go run .
```

Analyze your own text:

```bash
go run . "Hello Go"
```

Create and process multiple independent tasks:

```bash
go run . "Hello Go world" "Привет мир"
```

Each command-line argument is treated as one task. Use quotation marks to pass a text containing spaces.

Example output for `go run . "Hello Go"`:

```text
Created task 1: pending
Task 1: completed | words=2 | characters=8
```

Empty or whitespace-only input is rejected:

```bash
go run . "   "
```

The application prints an error and exits with a nonzero status.

## Analysis Rules

- Words are separated by whitespace using `strings.Fields`.
- Punctuation remains part of a word.
- Characters are counted as Unicode code points using `utf8.RuneCountInString`.
- Spaces and line breaks are included in the character count.
- Some compound emoji contain multiple Unicode code points.

## Task Model and Lifecycle

Each task contains a unique integer ID, input text, status, and an optional analysis result.

Successful processing follows these transitions:

```text
Pending -> Processing -> Completed
```

A new task starts as Pending and has no result.

Only Pending tasks can be processed. If processing returns an error, the task returns from Processing to Pending, allowing another attempt.

Completed tasks cannot be processed again. Retrieving or processing an unknown ID returns an error.

Tasks and results are stored in memory. IDs are unique within one service instance. Data is cleared when the application exits.

## Project Structure

- `task.go`: task statuses and data structures.
- `analyzer.go`: input validation and text analysis.
- `service.go`: in-memory storage, ID assignment, retrieval, and task lifecycle.
- `main.go`: command-line entry point.
- `analyzer_test.go`: table-driven tests for analysis and invalid input.
- `service_test.go`: tests for creation, retrieval, processing, errors, and independent tasks.

The service uses the Processor interface to call the text analyzer.

## Tests and Checks

```bash
gofmt -l .
go vet ./...
go test ./...
```

For detailed test output:

```bash
go test -v ./...
```

Tests cover task creation, input validation, retrieval, unknown IDs, processing results, completed status, invalid operations, unique IDs, and independent tasks.
