package lifecycle

import (
	"context"
	"time"

	"forge.lthn.ai/core/go-log"
)

// SubmitTask creates a new task with the given parameters via the API client.
// It validates that title is non-empty, sets CreatedAt to the current time,
// and delegates creation to client.CreateTask.
func SubmitTask(ctx context.Context, client *Client, title, description string, labels []string, priority TaskPriority) (*Task, error) {
	const op = "agentic.SubmitTask"

	if title == "" {
		return nil, log.E(op, "title is required", nil)
	}

	task := Task{
		Title:       title,
		Description: description,
		Labels:      labels,
		Priority:    priority,
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
	}

	created, err := client.CreateTask(ctx, task)
	if err != nil {
		return nil, log.E(op, "failed to create task", err)
	}

	return created, nil
}
