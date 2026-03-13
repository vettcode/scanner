// Package processor handles job processing
// Known complexity: ProcessJob=5, handleResult=4, validateJob=3
package processor

import (
	"errors"
	"fmt"
	"time"
)

// Job represents a work item.
type Job struct {
	ID       string
	Type     string
	Payload  map[string]interface{}
	Priority int
}

// Result represents processing outcome.
type Result struct {
	JobID   string
	Success bool
	Error   string
	Time    time.Duration
}

// ProcessJob executes a job based on its type.
// complexity: 1 (base) + 4 decision points = 5
func ProcessJob(job *Job) (*Result, error) {
	if job == nil {
		return nil, errors.New("nil job")
	}

	if err := validateJob(job); err != nil {
		return &Result{JobID: job.ID, Success: false, Error: err.Error()}, err
	}

	start := time.Now()

	switch job.Type {
	case "email":
		return &Result{JobID: job.ID, Success: true, Time: time.Since(start)}, nil
	case "report":
		return &Result{JobID: job.ID, Success: true, Time: time.Since(start)}, nil
	case "cleanup":
		return &Result{JobID: job.ID, Success: true, Time: time.Since(start)}, nil
	default:
		return nil, fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// handleResult processes a result and decides next steps.
// complexity: 1 (base) + 3 decision points = 4
func handleResult(result *Result) string {
	if result == nil {
		return "skip"
	}

	if !result.Success {
		if result.Error != "" {
			return "retry"
		}
		return "fail"
	}

	return "complete"
}

// validateJob checks job fields.
// complexity: 1 (base) + 2 decision points = 3
func validateJob(job *Job) error {
	if job.ID == "" {
		return errors.New("missing job ID")
	}
	if job.Type == "" {
		return errors.New("missing job type")
	}
	return nil
}
