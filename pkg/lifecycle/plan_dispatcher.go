package lifecycle

import (
	"context"
	"time"

	"forge.lthn.ai/core/go-log"
)

// PlanDispatcher orchestrates plan-based work by polling active plans,
// starting sessions, and routing work to agents. It wraps the existing
// agent registry, router, and allowance service alongside the API client.
type PlanDispatcher struct {
	registry  AgentRegistry
	router    TaskRouter
	allowance *AllowanceService
	client    *Client
	events    EventEmitter
	agentType string // e.g. "opus", "haiku", "codex"
}

// NewPlanDispatcher creates a PlanDispatcher for the given agent type.
func NewPlanDispatcher(
	agentType string,
	registry AgentRegistry,
	router TaskRouter,
	allowance *AllowanceService,
	client *Client,
) *PlanDispatcher {
	return &PlanDispatcher{
		agentType: agentType,
		registry:  registry,
		router:    router,
		allowance: allowance,
		client:    client,
	}
}

// SetEventEmitter attaches an event emitter for lifecycle notifications.
func (pd *PlanDispatcher) SetEventEmitter(em EventEmitter) {
	pd.events = em
}

func (pd *PlanDispatcher) emit(ctx context.Context, event Event) {
	if pd.events != nil {
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now().UTC()
		}
		_ = pd.events.Emit(ctx, event)
	}
}

// PlanDispatchLoop polls for active plans at the given interval and picks up
// the first plan with a pending or in-progress phase. It starts a session,
// marks the phase in-progress, and returns the plan + session for the caller
// to work on. Runs until context is cancelled.
func (pd *PlanDispatcher) PlanDispatchLoop(ctx context.Context, interval time.Duration) error {
	const op = "PlanDispatcher.PlanDispatchLoop"

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			plan, session, err := pd.pickUpWork(ctx)
			if err != nil {
				_ = log.E(op, "failed to pick up work", err)
				continue
			}
			if plan == nil {
				continue // no work available
			}

			pd.emit(ctx, Event{
				Type:    EventTaskDispatched,
				TaskID:  plan.Slug,
				AgentID: session.SessionID,
				Payload: map[string]string{
					"plan":       plan.Slug,
					"agent_type": pd.agentType,
				},
			})
		}
	}
}

// pickUpWork finds the first active plan with workable phases, starts a session,
// and marks the next phase in-progress. Returns nil if no work is available.
func (pd *PlanDispatcher) pickUpWork(ctx context.Context) (*Plan, *sessionStartResponse, error) {
	const op = "PlanDispatcher.pickUpWork"

	plans, err := pd.client.ListPlans(ctx, ListPlanOptions{Status: PlanActive})
	if err != nil {
		return nil, nil, log.E(op, "failed to list active plans", err)
	}

	for _, plan := range plans {
		// Check agent allowance before taking work.
		if pd.allowance != nil {
			check, err := pd.allowance.Check(pd.agentType, "")
			if err != nil || !check.Allowed {
				continue
			}
		}

		// Get full plan with phases.
		fullPlan, err := pd.client.GetPlan(ctx, plan.Slug)
		if err != nil {
			_ = log.E(op, "failed to get plan "+plan.Slug, err)
			continue
		}

		// Find the next workable phase.
		phase := nextWorkablePhase(fullPlan.Phases)
		if phase == nil {
			continue
		}

		// Start session for this plan.
		session, err := pd.client.StartSession(ctx, StartSessionRequest{
			AgentType: pd.agentType,
			PlanSlug:  plan.Slug,
		})
		if err != nil {
			_ = log.E(op, "failed to start session for "+plan.Slug, err)
			continue
		}

		// Mark phase as in-progress.
		if phase.Status == PhasePending {
			if err := pd.client.UpdatePhaseStatus(ctx, plan.Slug, phase.Name, PhaseInProgress, ""); err != nil {
				_ = log.E(op, "failed to update phase status", err)
			}
		}

		// Record job start.
		if pd.allowance != nil {
			_ = pd.allowance.RecordUsage(UsageReport{
				AgentID:   pd.agentType,
				JobID:     plan.Slug,
				Event:     QuotaEventJobStarted,
				Timestamp: time.Now().UTC(),
			})
		}

		return fullPlan, session, nil
	}

	return nil, nil, nil
}

// CompleteWork ends a session and optionally marks the current phase as completed.
func (pd *PlanDispatcher) CompleteWork(ctx context.Context, planSlug, sessionID, phaseName string, summary string) error {
	const op = "PlanDispatcher.CompleteWork"

	// Mark phase completed.
	if phaseName != "" {
		if err := pd.client.UpdatePhaseStatus(ctx, planSlug, phaseName, PhaseCompleted, ""); err != nil {
			_ = log.E(op, "failed to complete phase", err)
		}
	}

	// End session.
	if err := pd.client.EndSession(ctx, sessionID, "completed", summary); err != nil {
		return log.E(op, "failed to end session", err)
	}

	// Record job completion.
	if pd.allowance != nil {
		_ = pd.allowance.RecordUsage(UsageReport{
			AgentID:   pd.agentType,
			JobID:     planSlug,
			Event:     QuotaEventJobCompleted,
			Timestamp: time.Now().UTC(),
		})
	}

	return nil
}

// nextWorkablePhase returns the first phase that is pending or in-progress.
func nextWorkablePhase(phases []Phase) *Phase {
	for i := range phases {
		switch phases[i].Status {
		case PhasePending:
			if phases[i].CanStart {
				return &phases[i]
			}
		case PhaseInProgress:
			return &phases[i]
		}
	}
	return nil
}
