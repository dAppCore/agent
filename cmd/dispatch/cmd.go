package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"forge.lthn.ai/core/cli/pkg/cli"
	coreio "forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-log"

	agentic "forge.lthn.ai/core/agent/pkg/lifecycle"
)

func init() {
	cli.RegisterCommands(AddDispatchCommands)
}

// AddDispatchCommands registers the 'dispatch' subcommand group under 'ai'.
// These commands run ON the agent machine to process the work queue.
func AddDispatchCommands(parent *cli.Command) {
	dispatchCmd := &cli.Command{
		Use:   "dispatch",
		Short: "Agent work queue processor (runs on agent machine)",
	}

	dispatchCmd.AddCommand(dispatchRunCmd())
	dispatchCmd.AddCommand(dispatchWatchCmd())
	dispatchCmd.AddCommand(dispatchStatusCmd())

	parent.AddCommand(dispatchCmd)
}

// dispatchTicket represents the work item JSON structure.
type dispatchTicket struct {
	ID           string `json:"id"`
	RepoOwner    string `json:"repo_owner"`
	RepoName     string `json:"repo_name"`
	IssueNumber  int    `json:"issue_number"`
	IssueTitle   string `json:"issue_title"`
	IssueBody    string `json:"issue_body"`
	TargetBranch string `json:"target_branch"`
	EpicNumber   int    `json:"epic_number"`
	ForgeURL     string `json:"forge_url"`
	ForgeToken   string `json:"forge_token"`
	ForgeUser    string `json:"forgejo_user"`
	Model        string `json:"model"`
	Runner       string `json:"runner"`
	Timeout      string `json:"timeout"`
	CreatedAt    string `json:"created_at"`
}

const (
	defaultWorkDir = "ai-work"
	lockFileName   = ".runner.lock"
)

type runnerPaths struct {
	root   string
	queue  string
	active string
	done   string
	logs   string
	jobs   string
	lock   string
}

func getPaths(baseDir string) runnerPaths {
	if baseDir == "" {
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, defaultWorkDir)
	}
	return runnerPaths{
		root:   baseDir,
		queue:  filepath.Join(baseDir, "queue"),
		active: filepath.Join(baseDir, "active"),
		done:   filepath.Join(baseDir, "done"),
		logs:   filepath.Join(baseDir, "logs"),
		jobs:   filepath.Join(baseDir, "jobs"),
		lock:   filepath.Join(baseDir, lockFileName),
	}
}

func dispatchRunCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "run",
		Short: "Process a single ticket from the queue",
		RunE: func(cmd *cli.Command, args []string) error {
			workDir, _ := cmd.Flags().GetString("work-dir")
			paths := getPaths(workDir)

			if err := ensureDispatchDirs(paths); err != nil {
				return err
			}

			if err := acquireLock(paths.lock); err != nil {
				log.Info("Runner locked, skipping run", "lock", paths.lock)
				return nil
			}
			defer releaseLock(paths.lock)

			ticketFile, err := pickOldestTicket(paths.queue)
			if err != nil {
				return err
			}
			if ticketFile == "" {
				return nil
			}

			_, err = processTicket(paths, ticketFile)
			return err
		},
	}
	cmd.Flags().String("work-dir", "", "Working directory (default: ~/ai-work)")
	return cmd
}

// fastFailThreshold is how quickly a job must fail to be considered rate-limited.
// Real work always takes longer than 30 seconds; a 3-second exit means the CLI
// was rejected before it could start (rate limit, auth error, etc.).
const fastFailThreshold = 30 * time.Second

// maxBackoffMultiplier caps the exponential backoff at 8x the base interval.
const maxBackoffMultiplier = 8

func dispatchWatchCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "watch",
		Short: "Poll the PHP agentic API for work",
		RunE: func(cmd *cli.Command, args []string) error {
			workDir, _ := cmd.Flags().GetString("work-dir")
			interval, _ := cmd.Flags().GetDuration("interval")
			agentID, _ := cmd.Flags().GetString("agent-id")
			agentType, _ := cmd.Flags().GetString("agent-type")
			apiURL, _ := cmd.Flags().GetString("api-url")
			apiKey, _ := cmd.Flags().GetString("api-key")

			paths := getPaths(workDir)
			if err := ensureDispatchDirs(paths); err != nil {
				return err
			}

			// Create the go-agentic API client.
			client := agentic.NewClient(apiURL, apiKey)
			client.AgentID = agentID

			// Verify connectivity.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := client.Ping(ctx); err != nil {
				return log.E("dispatch.watch", "API ping failed (url="+apiURL+")", err)
			}
			log.Info("Connected to agentic API", "url", apiURL, "agent", agentID)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Backoff state.
			backoffMultiplier := 1
			currentInterval := interval

			ticker := time.NewTicker(currentInterval)
			defer ticker.Stop()

			adjustTicker := func(fastFail bool) {
				if fastFail {
					if backoffMultiplier < maxBackoffMultiplier {
						backoffMultiplier *= 2
					}
					currentInterval = interval * time.Duration(backoffMultiplier)
					log.Warn("Fast failure detected, backing off",
						"multiplier", backoffMultiplier, "next_poll", currentInterval)
				} else {
					if backoffMultiplier > 1 {
						log.Info("Job succeeded, resetting backoff")
					}
					backoffMultiplier = 1
					currentInterval = interval
				}
				ticker.Reset(currentInterval)
			}

			log.Info("Starting API poller", "interval", interval, "agent", agentID, "type", agentType)

			// Initial poll.
			ff := pollAndExecute(ctx, client, agentID, agentType, paths)
			adjustTicker(ff)

			for {
				select {
				case <-ticker.C:
					ff := pollAndExecute(ctx, client, agentID, agentType, paths)
					adjustTicker(ff)
				case <-sigChan:
					log.Info("Shutting down watcher...")
					return nil
				case <-ctx.Done():
					return nil
				}
			}
		},
	}
	cmd.Flags().String("work-dir", "", "Working directory (default: ~/ai-work)")
	cmd.Flags().Duration("interval", 2*time.Minute, "Polling interval")
	cmd.Flags().String("agent-id", defaultAgentID(), "Agent identifier")
	cmd.Flags().String("agent-type", "opus", "Agent type (opus, sonnet, gemini)")
	cmd.Flags().String("api-url", "https://api.lthn.sh", "Agentic API base URL")
	cmd.Flags().String("api-key", os.Getenv("AGENTIC_API_KEY"), "Agentic API key")
	return cmd
}

// pollAndExecute checks the API for workable plans and executes one phase per cycle.
// Returns true if a fast failure occurred (signals backoff).
func pollAndExecute(ctx context.Context, client *agentic.Client, agentID, agentType string, paths runnerPaths) bool {
	// List active plans.
	plans, err := client.ListPlans(ctx, agentic.ListPlanOptions{Status: agentic.PlanActive})
	if err != nil {
		log.Error("Failed to list plans", "error", err)
		return false
	}
	if len(plans) == 0 {
		log.Debug("No active plans")
		return false
	}

	// Find the first workable phase across all plans.
	for _, plan := range plans {
		// Fetch full plan with phases.
		fullPlan, err := client.GetPlan(ctx, plan.Slug)
		if err != nil {
			log.Error("Failed to get plan", "slug", plan.Slug, "error", err)
			continue
		}

		// Find first workable phase.
		var targetPhase *agentic.Phase
		for i := range fullPlan.Phases {
			p := &fullPlan.Phases[i]
			switch p.Status {
			case agentic.PhaseInProgress:
				targetPhase = p
			case agentic.PhasePending:
				if p.CanStart {
					targetPhase = p
				}
			}
			if targetPhase != nil {
				break
			}
		}
		if targetPhase == nil {
			continue
		}

		log.Info("Found workable phase",
			"plan", fullPlan.Slug, "phase", targetPhase.Name, "status", targetPhase.Status)

		// Start session.
		session, err := client.StartSession(ctx, agentic.StartSessionRequest{
			AgentType: agentType,
			PlanSlug:  fullPlan.Slug,
			Context: map[string]any{
				"agent_id": agentID,
				"phase":    targetPhase.Name,
			},
		})
		if err != nil {
			log.Error("Failed to start session", "error", err)
			return false
		}
		log.Info("Session started", "session_id", session.SessionID)

		// Mark phase in-progress if pending.
		if targetPhase.Status == agentic.PhasePending {
			if err := client.UpdatePhaseStatus(ctx, fullPlan.Slug, targetPhase.Name, agentic.PhaseInProgress, ""); err != nil {
				log.Warn("Failed to mark phase in-progress", "error", err)
			}
		}

		// Extract repo info from plan metadata.
		fastFail := executePhaseWork(ctx, client, fullPlan, targetPhase, session.SessionID, paths)
		return fastFail
	}

	log.Debug("No workable phases found across active plans")
	return false
}

// executePhaseWork does the actual repo prep + agent run for a phase.
// Returns true if the execution was a fast failure.
func executePhaseWork(ctx context.Context, client *agentic.Client, plan *agentic.Plan, phase *agentic.Phase, sessionID string, paths runnerPaths) bool {
	// Extract repo metadata from the plan.
	meta, _ := plan.Metadata.(map[string]any)
	repoOwner, _ := meta["repo_owner"].(string)
	repoName, _ := meta["repo_name"].(string)
	issueNumFloat, _ := meta["issue_number"].(float64) // JSON numbers are float64
	issueNumber := int(issueNumFloat)

	forgeURL, _ := meta["forge_url"].(string)
	forgeToken, _ := meta["forge_token"].(string)
	forgeUser, _ := meta["forgejo_user"].(string)
	targetBranch, _ := meta["target_branch"].(string)
	runner, _ := meta["runner"].(string)
	model, _ := meta["model"].(string)
	timeout, _ := meta["timeout"].(string)

	if targetBranch == "" {
		targetBranch = "main"
	}
	if runner == "" {
		runner = "claude"
	}

	// Build a dispatchTicket from the metadata so existing functions work.
	t := dispatchTicket{
		ID:           fmt.Sprintf("%s-%s", plan.Slug, phase.Name),
		RepoOwner:    repoOwner,
		RepoName:     repoName,
		IssueNumber:  issueNumber,
		IssueTitle:   plan.Title,
		IssueBody:    phase.Description,
		TargetBranch: targetBranch,
		ForgeURL:     forgeURL,
		ForgeToken:   forgeToken,
		ForgeUser:    forgeUser,
		Model:        model,
		Runner:       runner,
		Timeout:      timeout,
	}

	if t.RepoOwner == "" || t.RepoName == "" {
		log.Error("Plan metadata missing repo_owner or repo_name", "plan", plan.Slug)
		_ = client.EndSession(ctx, sessionID, string(agentic.SessionFailed), "missing repo metadata")
		return false
	}

	// Prepare the repository.
	jobDir := filepath.Join(paths.jobs, fmt.Sprintf("%s-%s-%d", t.RepoOwner, t.RepoName, t.IssueNumber))
	repoDir := filepath.Join(jobDir, t.RepoName)
	if err := coreio.Local.EnsureDir(jobDir); err != nil {
		log.Error("Failed to create job dir", "error", err)
		_ = client.EndSession(ctx, sessionID, string(agentic.SessionFailed), fmt.Sprintf("mkdir failed: %v", err))
		return false
	}

	if err := prepareRepo(t, repoDir); err != nil {
		log.Error("Repo preparation failed", "error", err)
		_ = client.UpdatePhaseStatus(ctx, plan.Slug, phase.Name, agentic.PhaseBlocked, fmt.Sprintf("git setup failed: %v", err))
		_ = client.EndSession(ctx, sessionID, string(agentic.SessionFailed), fmt.Sprintf("repo prep failed: %v", err))
		return false
	}

	// Build prompt and run.
	prompt := buildPrompt(t)
	logFile := filepath.Join(paths.logs, fmt.Sprintf("%s-%s.log", plan.Slug, phase.Name))

	start := time.Now()
	success, exitCode, runErr := runAgent(t, prompt, repoDir, logFile)
	elapsed := time.Since(start)

	// Detect fast failure.
	if !success && elapsed < fastFailThreshold {
		log.Warn("Agent rejected fast, likely rate-limited",
			"elapsed", elapsed.Round(time.Second), "plan", plan.Slug, "phase", phase.Name)
		_ = client.EndSession(ctx, sessionID, string(agentic.SessionFailed), "fast failure — likely rate-limited")
		return true
	}

	// Report results.
	if success {
		_ = client.UpdatePhaseStatus(ctx, plan.Slug, phase.Name, agentic.PhaseCompleted,
			fmt.Sprintf("completed in %s", elapsed.Round(time.Second)))
		_ = client.EndSession(ctx, sessionID, string(agentic.SessionCompleted),
			fmt.Sprintf("Phase %q completed successfully (exit %d, %s)", phase.Name, exitCode, elapsed.Round(time.Second)))
	} else {
		note := fmt.Sprintf("failed with exit code %d after %s", exitCode, elapsed.Round(time.Second))
		if runErr != nil {
			note += fmt.Sprintf(": %v", runErr)
		}
		_ = client.UpdatePhaseStatus(ctx, plan.Slug, phase.Name, agentic.PhaseBlocked, note)
		_ = client.EndSession(ctx, sessionID, string(agentic.SessionFailed), note)
	}

	// Also report to Forge issue if configured.
	msg := fmt.Sprintf("Agent completed phase %q of plan %q. Exit code: %d.", phase.Name, plan.Slug, exitCode)
	if !success {
		msg = fmt.Sprintf("Agent failed phase %q of plan %q (exit code: %d).", phase.Name, plan.Slug, exitCode)
	}
	reportToForge(t, success, msg)

	log.Info("Phase complete", "plan", plan.Slug, "phase", phase.Name, "success", success, "elapsed", elapsed.Round(time.Second))
	return false
}

// defaultAgentID returns a sensible agent ID from hostname.
func defaultAgentID() string {
	host, _ := os.Hostname()
	if host == "" {
		return "unknown"
	}
	return host
}

// --- Legacy registry/heartbeat functions (replaced by PHP API poller) ---

// registerAgent creates a SQLite registry and registers this agent.
// DEPRECATED: The watch command now uses the PHP agentic API instead.
// Kept for reference; remove once the API poller is proven stable.
/*
func registerAgent(agentID string, paths runnerPaths) (agentic.AgentRegistry, agentic.EventEmitter, func()) {
	dbPath := filepath.Join(paths.root, "registry.db")
	registry, err := agentic.NewSQLiteRegistry(dbPath)
	if err != nil {
		log.Warn("Failed to create agent registry", "error", err, "path", dbPath)
		return nil, nil, nil
	}

	info := agentic.AgentInfo{
		ID:            agentID,
		Name:          agentID,
		Status:        agentic.AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
		MaxLoad:       1,
	}
	if err := registry.Register(info); err != nil {
		log.Warn("Failed to register agent", "error", err)
	} else {
		log.Info("Agent registered", "id", agentID)
	}

	events := agentic.NewChannelEmitter(64)

	// Drain events to log.
	go func() {
		for ev := range events.Events() {
			log.Debug("Event", "type", string(ev.Type), "task", ev.TaskID, "agent", ev.AgentID)
		}
	}()

	return registry, events, func() {
		events.Close()
	}
}
*/

// heartbeatLoop sends periodic heartbeats to keep the agent status fresh.
// DEPRECATED: Replaced by PHP API poller.
/*
func heartbeatLoop(ctx context.Context, registry agentic.AgentRegistry, agentID string, interval time.Duration) {
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = registry.Heartbeat(agentID)
		}
	}
}
*/

// runCycleWithEvents wraps runCycle with registry status updates and event emission.
// DEPRECATED: Replaced by pollAndExecute.
/*
func runCycleWithEvents(paths runnerPaths, registry agentic.AgentRegistry, events agentic.EventEmitter, agentID string) bool {
	if registry != nil {
		if agent, err := registry.Get(agentID); err == nil {
			agent.Status = agentic.AgentBusy
			_ = registry.Register(agent)
		}
	}

	fastFail := runCycle(paths)

	if registry != nil {
		if agent, err := registry.Get(agentID); err == nil {
			agent.Status = agentic.AgentAvailable
			agent.LastHeartbeat = time.Now().UTC()
			_ = registry.Register(agent)
		}
	}
	return fastFail
}
*/

func dispatchStatusCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "status",
		Short: "Show runner status",
		RunE: func(cmd *cli.Command, args []string) error {
			workDir, _ := cmd.Flags().GetString("work-dir")
			paths := getPaths(workDir)

			lockStatus := "IDLE"
			if data, err := coreio.Local.Read(paths.lock); err == nil {
				pidStr := strings.TrimSpace(data)
				pid, _ := strconv.Atoi(pidStr)
				if isProcessAlive(pid) {
					lockStatus = fmt.Sprintf("RUNNING (PID %d)", pid)
				} else {
					lockStatus = fmt.Sprintf("STALE (PID %d)", pid)
				}
			}

			countFiles := func(dir string) int {
				entries, _ := os.ReadDir(dir)
				count := 0
				for _, e := range entries {
					if !e.IsDir() && strings.HasPrefix(e.Name(), "ticket-") {
						count++
					}
				}
				return count
			}

			fmt.Println("=== Agent Dispatch Status ===")
			fmt.Printf("Work Dir: %s\n", paths.root)
			fmt.Printf("Status:   %s\n", lockStatus)
			fmt.Printf("Queue:    %d\n", countFiles(paths.queue))
			fmt.Printf("Active:   %d\n", countFiles(paths.active))
			fmt.Printf("Done:     %d\n", countFiles(paths.done))

			return nil
		},
	}
	cmd.Flags().String("work-dir", "", "Working directory (default: ~/ai-work)")
	return cmd
}

// runCycle picks and processes one ticket. Returns true if the job fast-failed
// (likely rate-limited), signalling the caller to back off.
func runCycle(paths runnerPaths) bool {
	if err := acquireLock(paths.lock); err != nil {
		log.Debug("Runner locked, skipping cycle")
		return false
	}
	defer releaseLock(paths.lock)

	ticketFile, err := pickOldestTicket(paths.queue)
	if err != nil {
		log.Error("Failed to pick ticket", "error", err)
		return false
	}
	if ticketFile == "" {
		return false // empty queue, no backoff needed
	}

	start := time.Now()
	success, err := processTicket(paths, ticketFile)
	elapsed := time.Since(start)

	if err != nil {
		log.Error("Failed to process ticket", "file", ticketFile, "error", err)
	}

	// Detect fast failure: job failed in under 30s → likely rate-limited.
	if !success && elapsed < fastFailThreshold {
		log.Warn("Job finished too fast, likely rate-limited",
			"elapsed", elapsed.Round(time.Second), "file", filepath.Base(ticketFile))
		return true
	}

	return false
}

// processTicket processes a single ticket. Returns (success, error).
// On fast failure the caller is responsible for detecting the timing and backing off.
// The ticket is moved active→done on completion, or active→queue on fast failure.
func processTicket(paths runnerPaths, ticketPath string) (bool, error) {
	fileName := filepath.Base(ticketPath)
	log.Info("Processing ticket", "file", fileName)

	activePath := filepath.Join(paths.active, fileName)
	if err := os.Rename(ticketPath, activePath); err != nil {
		return false, log.E("processTicket", "failed to move ticket to active", err)
	}

	data, err := coreio.Local.Read(activePath)
	if err != nil {
		return false, log.E("processTicket", "failed to read ticket", err)
	}
	var t dispatchTicket
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return false, log.E("processTicket", "failed to unmarshal ticket", err)
	}

	jobDir := filepath.Join(paths.jobs, fmt.Sprintf("%s-%s-%d", t.RepoOwner, t.RepoName, t.IssueNumber))
	repoDir := filepath.Join(jobDir, t.RepoName)
	if err := coreio.Local.EnsureDir(jobDir); err != nil {
		return false, err
	}

	if err := prepareRepo(t, repoDir); err != nil {
		reportToForge(t, false, fmt.Sprintf("Git setup failed: %v", err))
		moveToDone(paths, activePath, fileName)
		return false, err
	}

	prompt := buildPrompt(t)

	logFile := filepath.Join(paths.logs, fmt.Sprintf("%s-%s-%d.log", t.RepoOwner, t.RepoName, t.IssueNumber))
	start := time.Now()
	success, exitCode, runErr := runAgent(t, prompt, repoDir, logFile)
	elapsed := time.Since(start)

	// Fast failure: agent exited in <30s without success → likely rate-limited.
	// Requeue the ticket so it's retried after the backoff period.
	if !success && elapsed < fastFailThreshold {
		log.Warn("Agent rejected fast, requeuing ticket", "elapsed", elapsed.Round(time.Second), "file", fileName)
		requeuePath := filepath.Join(paths.queue, fileName)
		if err := os.Rename(activePath, requeuePath); err != nil {
			// Fallback: move to done if requeue fails.
			moveToDone(paths, activePath, fileName)
		}
		return false, runErr
	}

	msg := fmt.Sprintf("Agent completed work on #%d. Exit code: %d.", t.IssueNumber, exitCode)
	if !success {
		msg = fmt.Sprintf("Agent failed on #%d (exit code: %d). Check logs on agent machine.", t.IssueNumber, exitCode)
		if runErr != nil {
			msg += fmt.Sprintf(" Error: %v", runErr)
		}
	}
	reportToForge(t, success, msg)

	moveToDone(paths, activePath, fileName)
	log.Info("Ticket complete", "id", t.ID, "success", success, "elapsed", elapsed.Round(time.Second))
	return success, nil
}

func prepareRepo(t dispatchTicket, repoDir string) error {
	user := t.ForgeUser
	if user == "" {
		host, _ := os.Hostname()
		user = fmt.Sprintf("%s-%s", host, os.Getenv("USER"))
	}

	cleanURL := strings.TrimPrefix(t.ForgeURL, "https://")
	cleanURL = strings.TrimPrefix(cleanURL, "http://")
	cloneURL := fmt.Sprintf("https://%s:%s@%s/%s/%s.git", user, t.ForgeToken, cleanURL, t.RepoOwner, t.RepoName)

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		log.Info("Updating existing repo", "dir", repoDir)
		cmds := [][]string{
			{"git", "fetch", "origin"},
			{"git", "checkout", t.TargetBranch},
			{"git", "pull", "origin", t.TargetBranch},
		}
		for _, args := range cmds {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = repoDir
			if out, err := cmd.CombinedOutput(); err != nil {
				if args[1] == "checkout" {
					createCmd := exec.Command("git", "checkout", "-b", t.TargetBranch, "origin/"+t.TargetBranch)
					createCmd.Dir = repoDir
					if _, err2 := createCmd.CombinedOutput(); err2 == nil {
						continue
					}
				}
				return log.E("prepareRepo", "git command failed: "+string(out), err)
			}
		}
	} else {
		log.Info("Cloning repo", "url", t.RepoOwner+"/"+t.RepoName)
		cmd := exec.Command("git", "clone", "-b", t.TargetBranch, cloneURL, repoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return log.E("prepareRepo", "git clone failed: "+string(out), err)
		}
	}
	return nil
}

func buildPrompt(t dispatchTicket) string {
	return fmt.Sprintf(`You are working on issue #%d in %s/%s.

Title: %s

Description:
%s

The repo is cloned at the current directory on branch '%s'.
Create a feature branch from '%s', make minimal targeted changes, commit referencing #%d, and push.
Then create a PR targeting '%s' using the forgejo MCP tools or git push.`,
		t.IssueNumber, t.RepoOwner, t.RepoName,
		t.IssueTitle,
		t.IssueBody,
		t.TargetBranch,
		t.TargetBranch, t.IssueNumber,
		t.TargetBranch,
	)
}

func runAgent(t dispatchTicket, prompt, dir, logPath string) (bool, int, error) {
	timeout := 30 * time.Minute
	if t.Timeout != "" {
		if d, err := time.ParseDuration(t.Timeout); err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	model := t.Model
	if model == "" {
		model = "sonnet"
	}

	log.Info("Running agent", "runner", t.Runner, "model", model)

	// For Gemini runner, wrap with rate limiting.
	if t.Runner == "gemini" {
		return executeWithRateLimit(ctx, model, prompt, func() (bool, int, error) {
			return execAgent(ctx, t.Runner, model, prompt, dir, logPath)
		})
	}

	return execAgent(ctx, t.Runner, model, prompt, dir, logPath)
}

func execAgent(ctx context.Context, runner, model, prompt, dir, logPath string) (bool, int, error) {
	var cmd *exec.Cmd

	switch runner {
	case "codex":
		cmd = exec.CommandContext(ctx, "codex", "exec", "--full-auto", prompt)
	case "gemini":
		args := []string{"-p", "-", "-y", "-m", model}
		cmd = exec.CommandContext(ctx, "gemini", args...)
		cmd.Stdin = strings.NewReader(prompt)
	default: // claude
		cmd = exec.CommandContext(ctx, "claude", "-p", "--model", model, "--dangerously-skip-permissions", "--output-format", "text")
		cmd.Stdin = strings.NewReader(prompt)
	}

	cmd.Dir = dir

	f, err := os.Create(logPath)
	if err != nil {
		return false, -1, err
	}
	defer f.Close()

	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Run(); err != nil {
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return false, exitCode, err
	}

	return true, 0, nil
}

func reportToForge(t dispatchTicket, success bool, body string) {
	token := t.ForgeToken
	if token == "" {
		token = os.Getenv("FORGE_TOKEN")
	}
	if token == "" {
		log.Warn("No forge token available, skipping report")
		return
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments",
		strings.TrimSuffix(t.ForgeURL, "/"), t.RepoOwner, t.RepoName, t.IssueNumber)

	payload := map[string]string{"body": body}
	jsonBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error("Failed to create request", "err", err)
		return
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to report to Forge", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Warn("Forge reported error", "status", resp.Status)
	}
}

func moveToDone(paths runnerPaths, activePath, fileName string) {
	donePath := filepath.Join(paths.done, fileName)
	if err := os.Rename(activePath, donePath); err != nil {
		log.Error("Failed to move ticket to done", "err", err)
	}
}

func ensureDispatchDirs(p runnerPaths) error {
	dirs := []string{p.queue, p.active, p.done, p.logs, p.jobs}
	for _, d := range dirs {
		if err := coreio.Local.EnsureDir(d); err != nil {
			return log.E("ensureDispatchDirs", "mkdir "+d+" failed", err)
		}
	}
	return nil
}

func acquireLock(lockPath string) error {
	if data, err := coreio.Local.Read(lockPath); err == nil {
		pidStr := strings.TrimSpace(data)
		pid, _ := strconv.Atoi(pidStr)
		if isProcessAlive(pid) {
			return log.E("acquireLock", fmt.Sprintf("locked by PID %d", pid), nil)
		}
		log.Info("Removing stale lock", "pid", pid)
		_ = coreio.Local.Delete(lockPath)
	}

	return coreio.Local.Write(lockPath, fmt.Sprintf("%d", os.Getpid()))
}

func releaseLock(lockPath string) {
	_ = coreio.Local.Delete(lockPath)
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func pickOldestTicket(queueDir string) (string, error) {
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		return "", err
	}

	var tickets []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "ticket-") && strings.HasSuffix(e.Name(), ".json") {
			tickets = append(tickets, filepath.Join(queueDir, e.Name()))
		}
	}

	if len(tickets) == 0 {
		return "", nil
	}

	slices.Sort(tickets)
	return tickets[0], nil
}
