package lifecycle

import (
	"context"
	"fmt"
	"io"
	"time"

	"forge.lthn.ai/core/go-log"
)

// StreamLogs polls a task's status and writes updates to writer. It polls at
// the given interval until the task reaches a terminal state (completed or
// blocked) or the context is cancelled. Returns ctx.Err() on cancellation.
func StreamLogs(ctx context.Context, client *Client, taskID string, interval time.Duration, writer io.Writer) error {
	const op = "agentic.StreamLogs"

	if taskID == "" {
		return log.E(op, "task ID is required", nil)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			task, err := client.GetTask(ctx, taskID)
			if err != nil {
				// Write the error but continue polling -- transient failures
				// should not stop the stream.
				_, _ = fmt.Fprintf(writer, "[%s] Error: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), err)
				continue
			}

			line := fmt.Sprintf("[%s] Status: %s", time.Now().UTC().Format("2006-01-02 15:04:05"), task.Status)
			_, _ = fmt.Fprintln(writer, line)

			// Stop on terminal states.
			if task.Status == StatusCompleted || task.Status == StatusBlocked {
				return nil
			}
		}
	}
}
