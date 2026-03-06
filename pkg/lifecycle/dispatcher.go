package lifecycle

import (
	"cmp"
	"context"
	"slices"
	"time"

	"forge.lthn.ai/core/go-log"
)

const (
	// DefaultMaxRetries is the default number of dispatch attempts before dead-lettering.
	DefaultMaxRetries = 3
	// baseBackoff is the base duration for exponential backoff between retries.
	baseBackoff = 5 * time.Second
)

// Dispatcher orchestrates task dispatch by combining the agent registry,
// task router, allowance service, and API client.
type Dispatcher struct {
	registry  AgentRegistry
	router    TaskRouter
	allowance *AllowanceService
	client    *Client // can be nil for tests
	events    EventEmitter
}

// NewDispatcher creates a new Dispatcher with the given dependencies.
func NewDispatcher(registry AgentRegistry, router TaskRouter, allowance *AllowanceService, client *Client) *Dispatcher {
	return &Dispatcher{
		registry:  registry,
		router:    router,
		allowance: allowance,
		client:    client,
	}
}

// SetEventEmitter attaches an event emitter to the dispatcher for lifecycle notifications.
func (d *Dispatcher) SetEventEmitter(em EventEmitter) {
	d.events = em
}

// emit is a convenience helper that publishes an event if an emitter is set.
func (d *Dispatcher) emit(ctx context.Context, event Event) {
	if d.events != nil {
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now().UTC()
		}
		_ = d.events.Emit(ctx, event)
	}
}

// Dispatch assigns a task to the best available agent. It queries the registry
// for available agents, routes the task, checks the agent's allowance, claims
// the task via the API client (if present), and records usage. Returns the
// assigned agent ID.
func (d *Dispatcher) Dispatch(ctx context.Context, task *Task) (string, error) {
	const op = "Dispatcher.Dispatch"

	// 1. Get available agents from registry.
	agents := d.registry.List()

	// 2. Route task to best agent.
	agentID, err := d.router.Route(task, agents)
	if err != nil {
		d.emit(ctx, Event{
			Type:   EventDispatchFailedNoAgent,
			TaskID: task.ID,
		})
		return "", log.E(op, "routing failed", err)
	}

	// 3. Check allowance for the selected agent.
	check, err := d.allowance.Check(agentID, "")
	if err != nil {
		return "", log.E(op, "allowance check failed", err)
	}
	if !check.Allowed {
		d.emit(ctx, Event{
			Type:    EventDispatchFailedQuota,
			TaskID:  task.ID,
			AgentID: agentID,
			Payload: check.Reason,
		})
		return "", log.E(op, "agent quota exceeded: "+check.Reason, nil)
	}

	// 4. Claim the task via the API client (if available).
	if d.client != nil {
		if _, err := d.client.ClaimTask(ctx, task.ID); err != nil {
			return "", log.E(op, "failed to claim task", err)
		}
		d.emit(ctx, Event{
			Type:    EventTaskClaimed,
			TaskID:  task.ID,
			AgentID: agentID,
		})
	}

	// 5. Record job start usage.
	if err := d.allowance.RecordUsage(UsageReport{
		AgentID:   agentID,
		JobID:     task.ID,
		Event:     QuotaEventJobStarted,
		Timestamp: time.Now().UTC(),
	}); err != nil {
		return "", log.E(op, "failed to record usage", err)
	}

	d.emit(ctx, Event{
		Type:    EventTaskDispatched,
		TaskID:  task.ID,
		AgentID: agentID,
	})

	return agentID, nil
}

// priorityRank maps a TaskPriority to a numeric rank for sorting.
// Lower values are dispatched first.
func priorityRank(p TaskPriority) int {
	switch p {
	case PriorityCritical:
		return 0
	case PriorityHigh:
		return 1
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 3
	default:
		return 4
	}
}

// sortTasksByPriority sorts tasks by priority (Critical first) then by
// CreatedAt (oldest first) as a tie-breaker. Uses slices.SortStableFunc for determinism.
func sortTasksByPriority(tasks []Task) {
	slices.SortStableFunc(tasks, func(a, b Task) int {
		ri, rj := priorityRank(a.Priority), priorityRank(b.Priority)
		if ri != rj {
			return cmp.Compare(ri, rj)
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})
}

// backoffDuration returns the exponential backoff duration for the given retry
// count. First retry waits baseBackoff (5s), second waits 10s, third 20s, etc.
func backoffDuration(retryCount int) time.Duration {
	if retryCount <= 0 {
		return 0
	}
	d := baseBackoff
	for range retryCount - 1 {
		d *= 2
	}
	return d
}

// shouldSkipRetry returns true if a task has been retried and the backoff
// period has not yet elapsed since the last attempt.
func shouldSkipRetry(task *Task, now time.Time) bool {
	if task.RetryCount <= 0 {
		return false
	}
	if task.LastAttempt == nil {
		return false
	}
	return task.LastAttempt.Add(backoffDuration(task.RetryCount)).After(now)
}

// effectiveMaxRetries returns the max retries for a task, using DefaultMaxRetries
// when the task does not specify one.
func effectiveMaxRetries(task *Task) int {
	if task.MaxRetries > 0 {
		return task.MaxRetries
	}
	return DefaultMaxRetries
}

// DispatchLoop polls for pending tasks at the given interval and dispatches
// each one. Tasks are sorted by priority (Critical > High > Medium > Low) with
// oldest-first tie-breaking. Failed dispatches are retried with exponential
// backoff. Tasks exceeding their retry limit are dead-lettered with StatusFailed.
// It runs until the context is cancelled and returns ctx.Err().
func (d *Dispatcher) DispatchLoop(ctx context.Context, interval time.Duration) error {
	const op = "Dispatcher.DispatchLoop"

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if d.client == nil {
				continue
			}

			tasks, err := d.client.ListTasks(ctx, ListOptions{Status: StatusPending})
			if err != nil {
				// Log but continue — transient API errors should not stop the loop.
				_ = log.E(op, "failed to list pending tasks", err)
				continue
			}

			// Sort by priority then by creation time.
			sortTasksByPriority(tasks)

			now := time.Now().UTC()
			for i := range tasks {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				task := &tasks[i]

				// Check if backoff period has not elapsed for retried tasks.
				if shouldSkipRetry(task, now) {
					continue
				}

				if _, err := d.Dispatch(ctx, task); err != nil {
					// Increment retry count and record the attempt time.
					task.RetryCount++
					attemptTime := now
					task.LastAttempt = &attemptTime

					maxRetries := effectiveMaxRetries(task)
					if task.RetryCount >= maxRetries {
						// Dead-letter: mark as failed via the API.
						if updateErr := d.client.UpdateTask(ctx, task.ID, TaskUpdate{
							Status: StatusFailed,
							Notes:  "max retries exceeded",
						}); updateErr != nil {
							_ = log.E(op, "failed to dead-letter task "+task.ID, updateErr)
						}
						d.emit(ctx, Event{
							Type:    EventTaskDeadLettered,
							TaskID:  task.ID,
							Payload: "max retries exceeded",
						})
					} else {
						_ = log.E(op, "failed to dispatch task "+task.ID, err)
					}
				}
			}
		}
	}
}
