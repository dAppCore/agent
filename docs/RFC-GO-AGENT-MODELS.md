# core-agent — Models

> Structs, interfaces, and types extracted from source by Codex.
> Packages: agentic, brain, lib, messages, monitor, setup.

## agentic

**Import:** `dappco.re/go/agent/pkg/agentic`
**Files:** 27

Package agentic provides MCP tools for agent orchestration.
Prepares workspaces and dispatches subagents.

## Types

### AgentsConfig
- **File:** queue.go
- **Purpose:** AgentsConfig is the root of config/agent.yaml.
- **Fields:**
  - `Version int` — Configuration version number.
  - `Dispatch DispatchConfig` — Dispatch-specific configuration.
  - `Concurrency map[string]ConcurrencyLimit` — Per-pool concurrency settings.
  - `Rates map[string]RateConfig` — Per-pool rate-limit configuration.

### BlockedInfo
- **File:** status.go
- **Purpose:** BlockedInfo shows a workspace that needs human input.
- **Fields:**
  - `Name string` — Name of the item.
  - `Repo string` — Repository name.
  - `Agent string` — Agent name or pool identifier.
  - `Question string` — Blocking question that needs an answer.

### ChildRef
- **File:** epic.go
- **Purpose:** ChildRef references a child issue.
- **Fields:**
  - `Number int` — Numeric identifier.
  - `Title string` — Title text.
  - `URL string` — URL for the item.

### CompletionEvent
- **File:** events.go
- **Purpose:** CompletionEvent is emitted when a dispatched agent finishes. Written to ~/.core/workspace/events.jsonl as append-only log.
- **Fields:**
  - `Type string` — Type discriminator.
  - `Agent string` — Agent name or pool identifier.
  - `Workspace string` — Workspace identifier or path.
  - `Status string` — Current status string.
  - `Timestamp string` — Timestamp recorded for the event.

### ConcurrencyLimit
- **File:** queue.go
- **Purpose:** ConcurrencyLimit supports both flat (int) and nested (map with total + per-model) formats.
- **Fields:**
  - `Total int` — Total concurrent dispatches allowed for the pool.
  - `Models map[string]int` — Per-model concurrency caps.

### CreatePRInput
- **File:** pr.go
- **Purpose:** CreatePRInput is the input for agentic_create_pr.
- **Fields:**
  - `Workspace string` — workspace name (e.g. "mcp-1773581873")
  - `Title string` — PR title (default: task description)
  - `Body string` — PR body (default: auto-generated)
  - `Base string` — base branch (default: "main")
  - `DryRun bool` — preview without creating

### CreatePROutput
- **File:** pr.go
- **Purpose:** CreatePROutput is the output for agentic_create_pr.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `PRURL string` — Pull request URL.
  - `PRNum int` — Pull request number.
  - `Title string` — Title text.
  - `Branch string` — Branch name.
  - `Repo string` — Repository name.
  - `Pushed bool` — Whether changes were pushed upstream.

### DispatchConfig
- **File:** queue.go
- **Purpose:** DispatchConfig controls agent dispatch behaviour.
- **Fields:**
  - `DefaultAgent string` — Default agent used when one is not supplied.
  - `DefaultTemplate string` — Default prompt template slug.
  - `WorkspaceRoot string` — Root directory used for prepared workspaces.

### DispatchInput
- **File:** dispatch.go
- **Purpose:** DispatchInput is the input for agentic_dispatch.
- **Fields:**
  - `Repo string` — Target repo (e.g. "go-io")
  - `Org string` — Forge org (default "core")
  - `Task string` — What the agent should do
  - `Agent string` — "codex" (default), "claude", "gemini"
  - `Template string` — "conventions", "security", "coding" (default)
  - `PlanTemplate string` — Plan template slug
  - `Variables map[string]string` — Template variable substitution
  - `Persona string` — Persona slug
  - `Issue int` — Forge issue number → workspace: task-{num}/
  - `PR int` — PR number → workspace: pr-{num}/
  - `Branch string` — Branch → workspace: {branch}/
  - `Tag string` — Tag → workspace: {tag}/ (immutable)
  - `DryRun bool` — Preview without executing

### DispatchOutput
- **File:** dispatch.go
- **Purpose:** DispatchOutput is the output for agentic_dispatch.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Agent string` — Agent name or pool identifier.
  - `Repo string` — Repository name.
  - `WorkspaceDir string` — Workspace directory path.
  - `Prompt string` — Rendered prompt content.
  - `PID int` — Process ID for the spawned agent.
  - `OutputFile string` — Path to the captured process output file.

### DispatchSyncInput
- **File:** dispatch_sync.go
- **Purpose:** DispatchSyncInput is the input for a synchronous (blocking) task run.
- **Fields:**
  - `Org string` — Forge organisation or namespace.
  - `Repo string` — Repository name.
  - `Agent string` — Agent name or pool identifier.
  - `Task string` — Task description.
  - `Issue int` — Issue number.

### DispatchSyncResult
- **File:** dispatch_sync.go
- **Purpose:** DispatchSyncResult is the output of a synchronous task run.
- **Fields:**
  - `OK bool` — Whether the synchronous dispatch finished successfully.
  - `Status string` — Current status string.
  - `Error string` — Error message, if the operation failed.
  - `PRURL string` — Pull request URL.

### EpicInput
- **File:** epic.go
- **Purpose:** EpicInput is the input for agentic_create_epic.
- **Fields:**
  - `Repo string` — Target repo (e.g. "go-scm")
  - `Org string` — Forge org (default "core")
  - `Title string` — Epic title
  - `Body string` — Epic description (above checklist)
  - `Tasks []string` — Sub-task titles (become child issues)
  - `Labels []string` — Labels for epic + children (e.g. ["agentic"])
  - `Dispatch bool` — Auto-dispatch agents to each child
  - `Agent string` — Agent type for dispatch (default "claude")
  - `Template string` — Prompt template for dispatch (default "coding")

### EpicOutput
- **File:** epic.go
- **Purpose:** EpicOutput is the output for agentic_create_epic.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `EpicNumber int` — Epic issue number.
  - `EpicURL string` — Epic issue URL.
  - `Children []ChildRef` — Child issues created under the epic.
  - `Dispatched int` — Number of child issues dispatched to agents.

### ListPRsInput
- **File:** pr.go
- **Purpose:** ListPRsInput is the input for agentic_list_prs.
- **Fields:**
  - `Org string` — forge org (default "core")
  - `Repo string` — specific repo, or empty for all
  - `State string` — "open" (default), "closed", "all"
  - `Limit int` — max results (default 20)

### ListPRsOutput
- **File:** pr.go
- **Purpose:** ListPRsOutput is the output for agentic_list_prs.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Count int` — Number of pull requests returned.
  - `PRs []PRInfo` — Pull requests returned by the query.

### MirrorInput
- **File:** mirror.go
- **Purpose:** MirrorInput is the input for agentic_mirror.
- **Fields:**
  - `Repo string` — Specific repo, or empty for all
  - `DryRun bool` — Preview without pushing
  - `MaxFiles int` — Max files per PR (default 50, CodeRabbit limit)

### MirrorOutput
- **File:** mirror.go
- **Purpose:** MirrorOutput is the output for agentic_mirror.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Synced []MirrorSync` — Repositories that were synchronised.
  - `Skipped []string` — Skipped items or skip reason, depending on context.
  - `Count int` — Number of repos included in the mirror result.

### MirrorSync
- **File:** mirror.go
- **Purpose:** MirrorSync records one repo sync.
- **Fields:**
  - `Repo string` — Repository name.
  - `CommitsAhead int` — Number of commits ahead of the mirror target.
  - `FilesChanged int` — Number of changed files included in the sync.
  - `PRURL string` — Pull request URL.
  - `Pushed bool` — Whether changes were pushed upstream.
  - `Skipped string` — Skipped items or skip reason, depending on context.

### PRInfo
- **File:** pr.go
- **Purpose:** PRInfo represents a pull request.
- **Fields:**
  - `Repo string` — Repository name.
  - `Number int` — Numeric identifier.
  - `Title string` — Title text.
  - `State string` — Current state value.
  - `Author string` — Pull request author name.
  - `Branch string` — Branch name.
  - `Base string` — Base branch for the pull request.
  - `Labels []string` — Label names applied to the issue or pull request.
  - `Mergeable bool` — Whether Forge reports the PR as mergeable.
  - `URL string` — URL for the item.

### Phase
- **File:** plan.go
- **Purpose:** Phase represents a phase within an implementation plan.
- **Fields:**
  - `Number int` — Numeric identifier.
  - `Name string` — Name of the item.
  - `Status string` — pending, in_progress, done
  - `Criteria []string` — Acceptance criteria for the phase.
  - `Tests int` — Expected test count for the phase.
  - `Notes string` — Free-form notes attached to the object.

### Plan
- **File:** plan.go
- **Purpose:** Plan represents an implementation plan for agent work.
- **Fields:**
  - `ID string` — Stable identifier.
  - `Title string` — Title text.
  - `Status string` — draft, ready, in_progress, needs_verification, verified, approved
  - `Repo string` — Repository name.
  - `Org string` — Forge organisation or namespace.
  - `Objective string` — Plan objective.
  - `Phases []Phase` — Plan phases.
  - `Notes string` — Free-form notes attached to the object.
  - `Agent string` — Agent name or pool identifier.
  - `CreatedAt time.Time` — Creation timestamp.
  - `UpdatedAt time.Time` — Last-update timestamp.

### PlanCreateInput
- **File:** plan.go
- **Purpose:** PlanCreateInput is the input for agentic_plan_create.
- **Fields:**
  - `Title string` — Title text.
  - `Objective string` — Plan objective.
  - `Repo string` — Repository name.
  - `Org string` — Forge organisation or namespace.
  - `Phases []Phase` — Plan phases.
  - `Notes string` — Free-form notes attached to the object.

### PlanCreateOutput
- **File:** plan.go
- **Purpose:** PlanCreateOutput is the output for agentic_plan_create.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `ID string` — Stable identifier.
  - `Path string` — Filesystem path for the generated or stored item.

### PlanDeleteInput
- **File:** plan.go
- **Purpose:** PlanDeleteInput is the input for agentic_plan_delete.
- **Fields:**
  - `ID string` — Stable identifier.

### PlanDeleteOutput
- **File:** plan.go
- **Purpose:** PlanDeleteOutput is the output for agentic_plan_delete.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Deleted string` — Identifier of the deleted plan.

### PlanListInput
- **File:** plan.go
- **Purpose:** PlanListInput is the input for agentic_plan_list.
- **Fields:**
  - `Status string` — Current status string.
  - `Repo string` — Repository name.

### PlanListOutput
- **File:** plan.go
- **Purpose:** PlanListOutput is the output for agentic_plan_list.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Count int` — Number of plans returned.
  - `Plans []Plan` — Plans returned by the query.

### PlanReadInput
- **File:** plan.go
- **Purpose:** PlanReadInput is the input for agentic_plan_read.
- **Fields:**
  - `ID string` — Stable identifier.

### PlanReadOutput
- **File:** plan.go
- **Purpose:** PlanReadOutput is the output for agentic_plan_read.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Plan Plan` — Returned plan data.

### PlanUpdateInput
- **File:** plan.go
- **Purpose:** PlanUpdateInput is the input for agentic_plan_update.
- **Fields:**
  - `ID string` — Stable identifier.
  - `Status string` — Current status string.
  - `Title string` — Title text.
  - `Objective string` — Plan objective.
  - `Phases []Phase` — Plan phases.
  - `Notes string` — Free-form notes attached to the object.
  - `Agent string` — Agent name or pool identifier.

### PlanUpdateOutput
- **File:** plan.go
- **Purpose:** PlanUpdateOutput is the output for agentic_plan_update.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Plan Plan` — Returned plan data.

### PrepInput
- **File:** prep.go
- **Purpose:** PrepInput is the input for agentic_prep_workspace. One of Issue, PR, Branch, or Tag is required.
- **Fields:**
  - `Repo string` — required: e.g. "go-io"
  - `Org string` — default "core"
  - `Task string` — task description
  - `Agent string` — agent type
  - `Issue int` — Forge issue → workspace: task-{num}/
  - `PR int` — PR number → workspace: pr-{num}/
  - `Branch string` — branch → workspace: {branch}/
  - `Tag string` — tag → workspace: {tag}/ (immutable)
  - `Template string` — prompt template slug
  - `PlanTemplate string` — plan template slug
  - `Variables map[string]string` — template variable substitution
  - `Persona string` — persona slug
  - `DryRun bool` — preview without executing

### PrepOutput
- **File:** prep.go
- **Purpose:** PrepOutput is the output for agentic_prep_workspace.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `WorkspaceDir string` — Workspace directory path.
  - `RepoDir string` — Local repository checkout directory.
  - `Branch string` — Branch name.
  - `Prompt string` — Rendered prompt content.
  - `Memories int` — Number of recalled memories injected into the prompt.
  - `Consumers int` — Number of dependent modules or consumers discovered.
  - `Resumed bool` — Whether the workspace was resumed instead of freshly prepared.

### PrepSubsystem
- **File:** prep.go
- **Purpose:** PrepSubsystem provides agentic MCP tools for workspace orchestration. Agent lifecycle events are broadcast via c.ACTION(messages.AgentCompleted{}).
- **Fields:**
  - `core *core.Core` — Core framework instance for IPC, Config, Lock
  - `forge *forge.Forge` — Forge client used for issue, PR, and repository operations.
  - `forgeURL string` — Forge base URL.
  - `forgeToken string` — Forge API token.
  - `brainURL string` — OpenBrain API base URL.
  - `brainKey string` — OpenBrain API key.
  - `codePath string` — Local code root used for prepared workspaces.
  - `client *http.Client` — HTTP client used for remote and Forge requests.
  - `drainMu sync.Mutex` — Mutex guarding queue-drain operations.
  - `pokeCh chan struct{}` — Channel used to wake the queue runner.
  - `frozen bool` — Whether queue processing is frozen during shutdown.
  - `backoff map[string]time.Time` — pool → paused until
  - `failCount map[string]int` — pool → consecutive fast failures

### RateConfig
- **File:** queue.go
- **Purpose:** RateConfig controls pacing between task dispatches.
- **Fields:**
  - `ResetUTC string` — Daily quota reset time (UTC), e.g. "06:00"
  - `DailyLimit int` — Max requests per day (0 = unknown)
  - `MinDelay int` — Minimum seconds between task starts
  - `SustainedDelay int` — Delay when pacing for full-day use
  - `BurstWindow int` — Hours before reset where burst kicks in
  - `BurstDelay int` — Delay during burst window

### RateLimitInfo
- **File:** review_queue.go
- **Purpose:** RateLimitInfo tracks CodeRabbit rate limit state.
- **Fields:**
  - `Limited bool` — Whether the pool is currently rate-limited.
  - `RetryAt time.Time` — Time when the backoff expires.
  - `Message string` — Human-readable status message.

### RemoteDispatchInput
- **File:** remote.go
- **Purpose:** RemoteDispatchInput dispatches a task to a remote core-agent over HTTP.
- **Fields:**
  - `Host string` — Remote agent host (e.g. "charon", "10.69.69.165:9101")
  - `Repo string` — Target repo
  - `Task string` — What the agent should do
  - `Agent string` — Agent type (default: claude:opus)
  - `Template string` — Prompt template
  - `Persona string` — Persona slug
  - `Org string` — Forge org (default: core)
  - `Variables map[string]string` — Template variables

### RemoteDispatchOutput
- **File:** remote.go
- **Purpose:** RemoteDispatchOutput is the response from a remote dispatch.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Host string` — Remote host handling the request.
  - `Repo string` — Repository name.
  - `Agent string` — Agent name or pool identifier.
  - `WorkspaceDir string` — Workspace directory path.
  - `PID int` — Process ID for the spawned agent.
  - `Error string` — Error message, if the operation failed.

### RemoteStatusInput
- **File:** remote_status.go
- **Purpose:** RemoteStatusInput queries a remote core-agent for workspace status.
- **Fields:**
  - `Host string` — Remote agent host (e.g. "charon")

### RemoteStatusOutput
- **File:** remote_status.go
- **Purpose:** RemoteStatusOutput is the response from a remote status check.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Host string` — Remote host handling the request.
  - `Stats StatusOutput` — Status snapshot returned by the remote host.
  - `Error string` — Error message, if the operation failed.

### ResumeInput
- **File:** resume.go
- **Purpose:** ResumeInput is the input for agentic_resume.
- **Fields:**
  - `Workspace string` — workspace name (e.g. "go-scm-1773581173")
  - `Answer string` — answer to the blocked question (written to ANSWER.md)
  - `Agent string` — override agent type (default: same as original)
  - `DryRun bool` — preview without executing

### ResumeOutput
- **File:** resume.go
- **Purpose:** ResumeOutput is the output for agentic_resume.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Workspace string` — Workspace identifier or path.
  - `Agent string` — Agent name or pool identifier.
  - `PID int` — Process ID for the spawned agent.
  - `OutputFile string` — Path to the captured process output file.
  - `Prompt string` — Rendered prompt content.

### ReviewQueueInput
- **File:** review_queue.go
- **Purpose:** ReviewQueueInput controls the review queue runner.
- **Fields:**
  - `Limit int` — Max PRs to process this run (default: 4)
  - `Reviewer string` — "coderabbit" (default), "codex", or "both"
  - `DryRun bool` — Preview without acting
  - `LocalOnly bool` — Run review locally, don't touch GitHub

### ReviewQueueOutput
- **File:** review_queue.go
- **Purpose:** ReviewQueueOutput reports what happened.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Processed []ReviewResult` — Review results that were processed.
  - `Skipped []string` — Skipped items or skip reason, depending on context.
  - `RateLimit *RateLimitInfo` — Rate-limit information, when present.

### ReviewResult
- **File:** review_queue.go
- **Purpose:** ReviewResult is the outcome of reviewing one repo.
- **Fields:**
  - `Repo string` — Repository name.
  - `Verdict string` — clean, findings, rate_limited, error
  - `Findings int` — Number of findings (0 = clean)
  - `Action string` — merged, fix_dispatched, skipped, waiting
  - `Detail string` — Additional detail about the review result.

### ScanInput
- **File:** scan.go
- **Purpose:** ScanInput is the input for agentic_scan.
- **Fields:**
  - `Org string` — default "core"
  - `Labels []string` — filter by labels (default: agentic, help-wanted, bug)
  - `Limit int` — max issues to return

### ScanIssue
- **File:** scan.go
- **Purpose:** ScanIssue is a single actionable issue.
- **Fields:**
  - `Repo string` — Repository name.
  - `Number int` — Numeric identifier.
  - `Title string` — Title text.
  - `Labels []string` — Label names applied to the issue or pull request.
  - `Assignee string` — Assignee.
  - `URL string` — URL for the item.

### ScanOutput
- **File:** scan.go
- **Purpose:** ScanOutput is the output for agentic_scan.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Count int` — Number of issues returned by the scan.
  - `Issues []ScanIssue` — Issues returned by the scan.

### ShutdownInput
- **File:** shutdown.go
- **Purpose:** ShutdownInput is the input for agentic_dispatch_shutdown.
- **Fields:** none

### ShutdownOutput
- **File:** shutdown.go
- **Purpose:** ShutdownOutput is the output for agentic_dispatch_shutdown.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Running int` — Running value.
  - `Queued int` — Number of queued items.
  - `Message string` — Human-readable status message.

### StatusInput
- **File:** status.go
- **Purpose:** StatusInput is the input for agentic_status.
- **Fields:**
  - `Workspace string` — specific workspace name, or empty for all
  - `Limit int` — max results (default 100)
  - `Status string` — filter: running, completed, failed, blocked

### StatusOutput
- **File:** status.go
- **Purpose:** StatusOutput is the output for agentic_status. Returns stats by default. Only blocked workspaces are listed (they need attention).
- **Fields:**
  - `Total int` — Total number of tracked workspaces.
  - `Running int` — Running value.
  - `Queued int` — Number of queued items.
  - `Completed int` — Number of completed items.
  - `Failed int` — Failed results.
  - `Blocked []BlockedInfo` — List of blocked values.

### WatchInput
- **File:** watch.go
- **Purpose:** WatchInput is the input for agentic_watch.
- **Fields:**
  - `Workspaces []string` — Workspaces to watch. If empty, watches all running/queued workspaces.
  - `PollInterval int` — PollInterval in seconds (default: 5)
  - `Timeout int` — Timeout in seconds (default: 600 = 10 minutes)

### WatchOutput
- **File:** watch.go
- **Purpose:** WatchOutput is the result when all watched workspaces complete.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Completed []WatchResult` — Number of completed items.
  - `Failed []WatchResult` — Failed results.
  - `Duration string` — Duration string for the event or backoff.

### WatchResult
- **File:** watch.go
- **Purpose:** WatchResult describes one completed workspace.
- **Fields:**
  - `Workspace string` — Workspace identifier or path.
  - `Agent string` — Agent name or pool identifier.
  - `Repo string` — Repository name.
  - `Status string` — Current status string.
  - `PRURL string` — Pull request URL.

### WorkspaceStatus
- **File:** status.go
- **Purpose:** WorkspaceStatus represents the current state of an agent workspace.
- **Fields:**
  - `Status string` — running, completed, blocked, failed
  - `Agent string` — gemini, claude, codex
  - `Repo string` — target repo
  - `Org string` — forge org (e.g. "core")
  - `Task string` — task description
  - `Branch string` — git branch name
  - `Issue int` — forge issue number
  - `PID int` — process ID (if running)
  - `StartedAt time.Time` — when dispatch started
  - `UpdatedAt time.Time` — last status change
  - `Question string` — from BLOCKED.md
  - `Runs int` — how many times dispatched/resumed
  - `PRURL string` — pull request URL (after PR created)

## Functions

### AgentName
- **File:** paths.go
- **Signature:** `func AgentName() string`
- **Purpose:** AgentName returns the name of this agent based on hostname. Checks AGENT_NAME env var first.

### CoreRoot
- **File:** paths.go
- **Signature:** `func CoreRoot() string`
- **Purpose:** CoreRoot returns the root directory for core ecosystem files. Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core.

### DefaultBranch
- **File:** paths.go
- **Signature:** `func DefaultBranch(repoDir string) string`
- **Purpose:** DefaultBranch detects the default branch of a repo (main, master, etc.).

### GitHubOrg
- **File:** paths.go
- **Signature:** `func GitHubOrg() string`
- **Purpose:** GitHubOrg returns the GitHub org for mirror operations.

### LocalFs
- **File:** paths.go
- **Signature:** `func LocalFs() *core.Fs`
- **Purpose:** LocalFs returns an unrestricted filesystem instance for use by other packages.

### NewPrep
- **File:** prep.go
- **Signature:** `func NewPrep() *PrepSubsystem`
- **Purpose:** NewPrep creates an agentic subsystem.

### PlansRoot
- **File:** paths.go
- **Signature:** `func PlansRoot() string`
- **Purpose:** PlansRoot returns the root directory for agent plans.

### ReadStatus
- **File:** status.go
- **Signature:** `func ReadStatus(wsDir string) (*WorkspaceStatus, error)`
- **Purpose:** ReadStatus parses the status.json in a workspace directory.

### Register
- **File:** register.go
- **Signature:** `func Register(c *core.Core) core.Result`
- **Purpose:** Register is the service factory for core.WithService. Returns the PrepSubsystem instance — WithService auto-discovers the name from the package path and registers it. Startable/Stoppable/HandleIPCEvents are auto-discovered by RegisterService.

### RegisterHandlers
- **File:** handlers.go
- **Signature:** `func RegisterHandlers(c *core.Core, s *PrepSubsystem)`
- **Purpose:** RegisterHandlers registers the post-completion pipeline as discrete IPC handlers. Each handler listens for a specific message and emits the next in the chain:

### WorkspaceRoot
- **File:** paths.go
- **Signature:** `func WorkspaceRoot() string`
- **Purpose:** WorkspaceRoot returns the root directory for agent workspaces. Checks CORE_WORKSPACE env var first, falls back to ~/Code/.core/workspace.

## Methods

### ConcurrencyLimit.UnmarshalYAML
- **File:** queue.go
- **Signature:** `func (*ConcurrencyLimit) UnmarshalYAML(value *yaml.Node) error`
- **Purpose:** UnmarshalYAML handles both int and map forms.

### PrepSubsystem.DispatchSync
- **File:** dispatch_sync.go
- **Signature:** `func (*PrepSubsystem) DispatchSync(ctx context.Context, input DispatchSyncInput) DispatchSyncResult`
- **Purpose:** DispatchSync preps a workspace, spawns the agent directly (no queue, no concurrency check), and blocks until the agent completes.

### PrepSubsystem.Name
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) Name() string`
- **Purpose:** Name implements mcp.Subsystem.

### PrepSubsystem.OnShutdown
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) OnShutdown(ctx context.Context) error`
- **Purpose:** OnShutdown implements core.Stoppable — freezes the queue.

### PrepSubsystem.OnStartup
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) OnStartup(ctx context.Context) error`
- **Purpose:** OnStartup implements core.Startable — starts the queue runner and registers commands.

### PrepSubsystem.Poke
- **File:** runner.go
- **Signature:** `func (*PrepSubsystem) Poke()`
- **Purpose:** Poke signals the runner to check the queue immediately. Non-blocking — if a poke is already pending, this is a no-op.

### PrepSubsystem.RegisterTools
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) RegisterTools(server *mcp.Server)`
- **Purpose:** RegisterTools implements mcp.Subsystem.

### PrepSubsystem.SetCore
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) SetCore(c *core.Core)`
- **Purpose:** SetCore wires the Core framework instance for IPC, Config, and Lock access.

### PrepSubsystem.Shutdown
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) Shutdown(_ context.Context) error`
- **Purpose:** Shutdown implements mcp.SubsystemWithShutdown.

### PrepSubsystem.StartRunner
- **File:** runner.go
- **Signature:** `func (*PrepSubsystem) StartRunner()`
- **Purpose:** StartRunner begins the background queue runner. Queue is frozen by default — use agentic_dispatch_start to unfreeze, or set CORE_AGENT_DISPATCH=1 to auto-start.

### PrepSubsystem.TestBuildPrompt
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) TestBuildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int)`
- **Purpose:** TestBuildPrompt exposes buildPrompt for CLI testing.

### PrepSubsystem.TestPrepWorkspace
- **File:** prep.go
- **Signature:** `func (*PrepSubsystem) TestPrepWorkspace(ctx context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error)`
- **Purpose:** TestPrepWorkspace exposes prepWorkspace for CLI testing.


## brain

**Import:** `dappco.re/go/agent/pkg/brain`
**Files:** 6

Package brain provides an MCP subsystem that proxies OpenBrain knowledge
store operations to the Laravel php-agentic backend via the IDE bridge.

## Types

### BrainProvider
- **File:** provider.go
- **Purpose:** BrainProvider wraps the brain Subsystem as a service provider with REST endpoints. It delegates to the same IDE bridge that the MCP tools use.
- **Fields:**
  - `bridge *ide.Bridge` — IDE bridge used to access php-agentic services.
  - `hub *ws.Hub` — WebSocket hub exposed by the provider.

### ConversationInput
- **File:** messaging.go
- **Purpose:** ConversationInput selects the agent thread to load.
- **Fields:**
  - `Agent string` — Agent name or pool identifier.

### ConversationOutput
- **File:** messaging.go
- **Purpose:** ConversationOutput returns a direct message thread with another agent.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Messages []MessageItem` — Conversation or inbox messages.

### DirectSubsystem
- **File:** direct.go
- **Purpose:** DirectSubsystem calls the OpenBrain HTTP API without the IDE bridge.
- **Fields:**
  - `apiURL string` — Base URL for direct OpenBrain HTTP calls.
  - `apiKey string` — API key for direct OpenBrain HTTP calls.
  - `client *http.Client` — HTTP client used for direct requests.

### ForgetInput
- **File:** tools.go
- **Purpose:** ForgetInput is the input for brain_forget.
- **Fields:**
  - `ID string` — Stable identifier.
  - `Reason string` — Reason string supplied with the result.

### ForgetOutput
- **File:** tools.go
- **Purpose:** ForgetOutput is the output for brain_forget.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Forgotten string` — Identifier of the forgotten memory.
  - `Timestamp time.Time` — Timestamp recorded for the event.

### InboxInput
- **File:** messaging.go
- **Purpose:** InboxInput selects which agent inbox to read.
- **Fields:**
  - `Agent string` — Agent name or pool identifier.

### InboxOutput
- **File:** messaging.go
- **Purpose:** InboxOutput returns the latest direct messages for an agent.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Messages []MessageItem` — Conversation or inbox messages.

### ListInput
- **File:** tools.go
- **Purpose:** ListInput is the input for brain_list.
- **Fields:**
  - `Project string` — Project name associated with the request.
  - `Type string` — Type discriminator.
  - `AgentID string` — Agent identifier used by the brain service.
  - `Limit int` — Maximum number of items to return.

### ListOutput
- **File:** tools.go
- **Purpose:** ListOutput is the output for brain_list.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Count int` — Total number of returned items.
  - `Memories []Memory` — Returned memories or memory count, depending on context.

### Memory
- **File:** tools.go
- **Purpose:** Memory is a single memory entry returned by recall or list.
- **Fields:**
  - `ID string` — Stable identifier.
  - `AgentID string` — Agent identifier used by the brain service.
  - `Type string` — Type discriminator.
  - `Content string` — Message or memory content.
  - `Tags []string` — Tag values attached to the memory.
  - `Project string` — Project name associated with the request.
  - `Confidence float64` — Confidence score attached to the memory.
  - `SupersedesID string` — Identifier of the superseded memory.
  - `ExpiresAt string` — Expiration timestamp, when set.
  - `CreatedAt string` — Creation timestamp.
  - `UpdatedAt string` — Last-update timestamp.

### MessageItem
- **File:** messaging.go
- **Purpose:** MessageItem is one inbox or conversation message.
- **Fields:**
  - `ID int` — Stable identifier.
  - `From string` — Message sender.
  - `To string` — Message recipient.
  - `Subject string` — Message subject.
  - `Content string` — Message or memory content.
  - `Read bool` — Whether the message has been marked as read.
  - `CreatedAt string` — Creation timestamp.

### RecallFilter
- **File:** tools.go
- **Purpose:** RecallFilter holds optional filter criteria for brain_recall.
- **Fields:**
  - `Project string` — Project name associated with the request.
  - `Type any` — Type discriminator.
  - `AgentID string` — Agent identifier used by the brain service.
  - `MinConfidence float64` — Minimum confidence required when filtering recalls.

### RecallInput
- **File:** tools.go
- **Purpose:** RecallInput is the input for brain_recall.
- **Fields:**
  - `Query string` — Recall query text.
  - `TopK int` — Maximum number of recall matches to return.
  - `Filter RecallFilter` — Recall filter applied to the query.

### RecallOutput
- **File:** tools.go
- **Purpose:** RecallOutput is the output for brain_recall.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `Count int` — Total number of returned items.
  - `Memories []Memory` — Returned memories or memory count, depending on context.

### RememberInput
- **File:** tools.go
- **Purpose:** RememberInput is the input for brain_remember.
- **Fields:**
  - `Content string` — Message or memory content.
  - `Type string` — Type discriminator.
  - `Tags []string` — Tag values attached to the memory.
  - `Project string` — Project name associated with the request.
  - `Confidence float64` — Confidence score attached to the memory.
  - `Supersedes string` — Identifier of the memory this write supersedes.
  - `ExpiresIn int` — Relative expiry in seconds.

### RememberOutput
- **File:** tools.go
- **Purpose:** RememberOutput is the output for brain_remember.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `MemoryID string` — Identifier of the stored memory.
  - `Timestamp time.Time` — Timestamp recorded for the event.

### SendInput
- **File:** messaging.go
- **Purpose:** SendInput sends a direct message to another agent.
- **Fields:**
  - `To string` — Message recipient.
  - `Content string` — Message or memory content.
  - `Subject string` — Message subject.

### SendOutput
- **File:** messaging.go
- **Purpose:** SendOutput reports the created direct message.
- **Fields:**
  - `Success bool` — Whether the operation succeeded.
  - `ID int` — Stable identifier.
  - `To string` — Message recipient.

### Subsystem
- **File:** brain.go
- **Purpose:** Subsystem proxies brain_* MCP tools through the shared IDE bridge.
- **Fields:**
  - `bridge *ide.Bridge` — IDE bridge used to proxy requests into php-agentic.

## Functions

### New
- **File:** brain.go
- **Signature:** `func New(bridge *ide.Bridge) *Subsystem`
- **Purpose:** New creates a bridge-backed brain subsystem.

### NewDirect
- **File:** direct.go
- **Signature:** `func NewDirect() *DirectSubsystem`
- **Purpose:** NewDirect creates a direct HTTP brain subsystem.

### NewProvider
- **File:** provider.go
- **Signature:** `func NewProvider(bridge *ide.Bridge, hub *ws.Hub) *BrainProvider`
- **Purpose:** NewProvider creates a brain provider that proxies to Laravel via the IDE bridge. The WS hub is used to emit brain events. Pass nil for hub if not needed.

### Register
- **File:** register.go
- **Signature:** `func Register(c *core.Core) core.Result`
- **Purpose:** Register is the service factory for core.WithService. Returns the DirectSubsystem — WithService auto-registers it.

## Methods

### BrainProvider.BasePath
- **File:** provider.go
- **Signature:** `func (*BrainProvider) BasePath() string`
- **Purpose:** BasePath implements api.RouteGroup.

### BrainProvider.Channels
- **File:** provider.go
- **Signature:** `func (*BrainProvider) Channels() []string`
- **Purpose:** Channels implements provider.Streamable.

### BrainProvider.Describe
- **File:** provider.go
- **Signature:** `func (*BrainProvider) Describe() []api.RouteDescription`
- **Purpose:** Describe implements api.DescribableGroup.

### BrainProvider.Element
- **File:** provider.go
- **Signature:** `func (*BrainProvider) Element() provider.ElementSpec`
- **Purpose:** Element implements provider.Renderable.

### BrainProvider.Name
- **File:** provider.go
- **Signature:** `func (*BrainProvider) Name() string`
- **Purpose:** Name implements api.RouteGroup.

### BrainProvider.RegisterRoutes
- **File:** provider.go
- **Signature:** `func (*BrainProvider) RegisterRoutes(rg *gin.RouterGroup)`
- **Purpose:** RegisterRoutes implements api.RouteGroup.

### DirectSubsystem.Name
- **File:** direct.go
- **Signature:** `func (*DirectSubsystem) Name() string`
- **Purpose:** Name returns the MCP subsystem name.

### DirectSubsystem.RegisterMessagingTools
- **File:** messaging.go
- **Signature:** `func (*DirectSubsystem) RegisterMessagingTools(server *mcp.Server)`
- **Purpose:** RegisterMessagingTools adds direct agent messaging tools to an MCP server.

### DirectSubsystem.RegisterTools
- **File:** direct.go
- **Signature:** `func (*DirectSubsystem) RegisterTools(server *mcp.Server)`
- **Purpose:** RegisterTools adds the direct OpenBrain tools to an MCP server.

### DirectSubsystem.Shutdown
- **File:** direct.go
- **Signature:** `func (*DirectSubsystem) Shutdown(_ context.Context) error`
- **Purpose:** Shutdown closes the direct subsystem without additional cleanup.

### Subsystem.Name
- **File:** brain.go
- **Signature:** `func (*Subsystem) Name() string`
- **Purpose:** Name returns the MCP subsystem name.

### Subsystem.RegisterTools
- **File:** brain.go
- **Signature:** `func (*Subsystem) RegisterTools(server *mcp.Server)`
- **Purpose:** RegisterTools adds the bridge-backed brain tools to an MCP server.

### Subsystem.Shutdown
- **File:** brain.go
- **Signature:** `func (*Subsystem) Shutdown(_ context.Context) error`
- **Purpose:** Shutdown closes the subsystem without additional cleanup.


## lib

**Import:** `dappco.re/go/agent/pkg/lib`
**Files:** 1

Package lib provides embedded content for agent dispatch.
Prompts, tasks, flows, personas, and workspace templates.

Structure:

	prompt/      — System prompts (HOW to work)
	task/        — Structured task plans (WHAT to do)
	task/code/   — Code-specific tasks (review, refactor, etc.)
	flow/        — Build/release workflows per language/tool
	persona/     — Domain/role system prompts (WHO you are)
	workspace/   — Agent workspace templates (WHERE to work)

Usage:

	r := lib.Prompt("coding")        // r.Value.(string)
	r := lib.Task("code/review")     // r.Value.(string)
	r := lib.Persona("secops/dev")   // r.Value.(string)
	r := lib.Flow("go")              // r.Value.(string)
	lib.ExtractWorkspace("default", "/tmp/ws", data)

## Types

### Bundle
- **File:** lib.go
- **Purpose:** Bundle holds a task's main content plus companion files.
- **Fields:**
  - `Main string` — Primary bundled document content.
  - `Files map[string]string` — Number of files or bundled file contents, depending on context.

### WorkspaceData
- **File:** lib.go
- **Purpose:** WorkspaceData is the data passed to workspace templates.
- **Fields:**
  - `Repo string` — Repository name.
  - `Branch string` — Branch name.
  - `Task string` — Task description.
  - `Agent string` — Agent name or pool identifier.
  - `Language string` — Detected repository language.
  - `Prompt string` — Rendered prompt content.
  - `Persona string` — Persona slug injected into the workspace template.
  - `Flow string` — Workflow content or slug injected into the workspace template.
  - `Context string` — Additional context injected into a workspace template.
  - `Recent string` — Recent-change context injected into a workspace template.
  - `Dependencies string` — Dependency context injected into a workspace template.
  - `Conventions string` — Coding-convention guidance injected into a workspace template.
  - `RepoDescription string` — Repository description injected into the workspace template.
  - `BuildCmd string` — Build command injected into workspace templates.
  - `TestCmd string` — Test command injected into workspace templates.

## Functions

### ExtractWorkspace
- **File:** lib.go
- **Signature:** `func ExtractWorkspace(tmplName, targetDir string, data *WorkspaceData) error`
- **Purpose:** ExtractWorkspace creates an agent workspace from a template. Template names: "default", "security", "review".

### Flow
- **File:** lib.go
- **Signature:** `func Flow(slug string) core.Result`
- **Purpose:** Flow reads a build/release workflow by slug.

### ListFlows
- **File:** lib.go
- **Signature:** `func ListFlows() []string`
- **Purpose:** Lists embedded workflow slugs from the flow bundle.

### ListPersonas
- **File:** lib.go
- **Signature:** `func ListPersonas() []string`
- **Purpose:** Lists embedded persona paths from the persona bundle.

### ListPrompts
- **File:** lib.go
- **Signature:** `func ListPrompts() []string`
- **Purpose:** Lists embedded prompt slugs from the prompt bundle.

### ListTasks
- **File:** lib.go
- **Signature:** `func ListTasks() []string`
- **Purpose:** Lists embedded task slugs by walking the task bundle.

### ListWorkspaces
- **File:** lib.go
- **Signature:** `func ListWorkspaces() []string`
- **Purpose:** Lists embedded workspace template names from the workspace bundle.

### Persona
- **File:** lib.go
- **Signature:** `func Persona(path string) core.Result`
- **Purpose:** Persona reads a domain/role persona by path.

### Prompt
- **File:** lib.go
- **Signature:** `func Prompt(slug string) core.Result`
- **Purpose:** Prompt reads a system prompt by slug.

### Task
- **File:** lib.go
- **Signature:** `func Task(slug string) core.Result`
- **Purpose:** Task reads a structured task plan by slug. Tries .md, .yaml, .yml.

### TaskBundle
- **File:** lib.go
- **Signature:** `func TaskBundle(slug string) core.Result`
- **Purpose:** TaskBundle reads a task and its companion files.

### Template
- **File:** lib.go
- **Signature:** `func Template(slug string) core.Result`
- **Purpose:** Template tries Prompt then Task (backwards compat).

## Methods

No exported methods.


## messages

**Import:** `dappco.re/go/agent/pkg/messages`
**Files:** 1

Package messages defines IPC message types for inter-service communication
within core-agent. Services emit these via c.ACTION() and handle them via
c.RegisterAction(). No service imports another — they share only these types.

	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Status: "completed"})

## Types

### AgentCompleted
- **File:** messages.go
- **Purpose:** AgentCompleted is broadcast when a subagent process exits.
- **Fields:**
  - `Agent string` — Agent name or pool identifier.
  - `Repo string` — Repository name.
  - `Workspace string` — Workspace identifier or path.
  - `Status string` — completed, failed, blocked

### AgentStarted
- **File:** messages.go
- **Purpose:** AgentStarted is broadcast when a subagent process is spawned.
- **Fields:**
  - `Agent string` — Agent name or pool identifier.
  - `Repo string` — Repository name.
  - `Workspace string` — Workspace identifier or path.

### HarvestComplete
- **File:** messages.go
- **Purpose:** HarvestComplete is broadcast when a workspace branch is ready for review.
- **Fields:**
  - `Repo string` — Repository name.
  - `Branch string` — Branch name.
  - `Files int` — Number of files or bundled file contents, depending on context.

### HarvestRejected
- **File:** messages.go
- **Purpose:** HarvestRejected is broadcast when a workspace fails safety checks (binaries, size).
- **Fields:**
  - `Repo string` — Repository name.
  - `Branch string` — Branch name.
  - `Reason string` — Reason string supplied with the result.

### InboxMessage
- **File:** messages.go
- **Purpose:** InboxMessage is broadcast when new inter-agent messages arrive.
- **Fields:**
  - `New int` — Number of newly observed messages.
  - `Total int` — Total number of items observed.

### PRCreated
- **File:** messages.go
- **Purpose:** PRCreated is broadcast after a PR is auto-created on Forge.
- **Fields:**
  - `Repo string` — Repository name.
  - `Branch string` — Branch name.
  - `PRURL string` — Pull request URL.
  - `PRNum int` — Pull request number.

### PRMerged
- **File:** messages.go
- **Purpose:** PRMerged is broadcast after a PR is auto-verified and merged.
- **Fields:**
  - `Repo string` — Repository name.
  - `PRURL string` — Pull request URL.
  - `PRNum int` — Pull request number.

### PRNeedsReview
- **File:** messages.go
- **Purpose:** PRNeedsReview is broadcast when auto-merge fails and human attention is needed.
- **Fields:**
  - `Repo string` — Repository name.
  - `PRURL string` — Pull request URL.
  - `PRNum int` — Pull request number.
  - `Reason string` — Reason string supplied with the result.

### PokeQueue
- **File:** messages.go
- **Purpose:** PokeQueue signals the runner to drain the queue immediately.
- **Fields:** none

### QAResult
- **File:** messages.go
- **Purpose:** QAResult is broadcast after QA runs on a completed workspace.
- **Fields:**
  - `Workspace string` — Workspace identifier or path.
  - `Repo string` — Repository name.
  - `Passed bool` — Whether QA passed.
  - `Output string` — Command output or QA output text.

### QueueDrained
- **File:** messages.go
- **Purpose:** QueueDrained is broadcast when running=0 and queued=0 (genuinely empty).
- **Fields:**
  - `Completed int` — Number of completed items.

### RateLimitDetected
- **File:** messages.go
- **Purpose:** RateLimitDetected is broadcast when fast failures trigger agent pool backoff.
- **Fields:**
  - `Pool string` — Agent pool that triggered the event.
  - `Duration string` — Duration string for the event or backoff.

## Functions

No exported functions.

## Methods

No exported methods.


## monitor

**Import:** `dappco.re/go/agent/pkg/monitor`
**Files:** 4

Package monitor provides a background subsystem that watches the ecosystem
and pushes notifications to connected MCP clients.

Checks performed on each tick:
  - Agent completions: scans workspace for newly completed agents
  - Repo drift: checks forge for repos with unpushed/unpulled changes
  - Inbox: checks for unread agent messages

## Types

### ChangedRepo
- **File:** sync.go
- **Purpose:** ChangedRepo is a repo that has new commits.
- **Fields:**
  - `Repo string` — Repository name.
  - `Branch string` — Branch name.
  - `SHA string` — Commit SHA.

### ChannelNotifier
- **File:** monitor.go
- **Purpose:** ChannelNotifier pushes events to connected MCP sessions.
- **Methods:**
  - `ChannelSend(ctx context.Context, channel string, data any)` — Sends a payload to a named notifier channel.

### CheckinResponse
- **File:** sync.go
- **Purpose:** CheckinResponse is what the API returns for an agent checkin.
- **Fields:**
  - `Changed []ChangedRepo` — Repos that have new commits since the agent's last checkin.
  - `Timestamp int64` — Server timestamp — use as "since" on next checkin.

### Options
- **File:** monitor.go
- **Purpose:** Options configures the monitor interval.
- **Fields:**
  - `Interval time.Duration` — Interval between checks (default: 2 minutes)

### Subsystem
- **File:** monitor.go
- **Purpose:** Subsystem implements mcp.Subsystem for background monitoring.
- **Fields:**
  - `core *core.Core` — Core framework instance for IPC
  - `server *mcp.Server` — MCP server used to register monitor resources.
  - `notifier ChannelNotifier` — Channel notification relay, uses c.ACTION()
  - `interval time.Duration` — Interval between monitor scans.
  - `cancel context.CancelFunc` — Cancellation function for the monitor loop.
  - `wg sync.WaitGroup` — WaitGroup tracking monitor goroutines.
  - `lastCompletedCount int` — Track last seen state to only notify on changes
  - `seenCompleted map[string]bool` — workspace names we've already notified about
  - `seenRunning map[string]bool` — workspace names we've already sent start notification for
  - `completionsSeeded bool` — true after first completions check
  - `lastInboxMaxID int` — highest message ID seen
  - `inboxSeeded bool` — true after first inbox check
  - `lastSyncTimestamp int64` — Unix timestamp of the last repo-sync check.
  - `mu sync.Mutex` — Mutex guarding monitor state.
  - `poke chan struct{}` — Event-driven poke channel — dispatch goroutine sends here on completion

## Functions

### New
- **File:** monitor.go
- **Signature:** `func New(opts ...Options) *Subsystem`
- **Purpose:** New creates a monitor subsystem.

### Register
- **File:** register.go
- **Signature:** `func Register(c *core.Core) core.Result`
- **Purpose:** Register is the service factory for core.WithService. Returns the monitor Subsystem — WithService auto-registers it.

## Methods

### Subsystem.Name
- **File:** monitor.go
- **Signature:** `func (*Subsystem) Name() string`
- **Purpose:** Name returns the subsystem identifier used by MCP registration.

### Subsystem.OnShutdown
- **File:** monitor.go
- **Signature:** `func (*Subsystem) OnShutdown(ctx context.Context) error`
- **Purpose:** OnShutdown implements core.Stoppable — stops the monitoring loop.

### Subsystem.OnStartup
- **File:** monitor.go
- **Signature:** `func (*Subsystem) OnStartup(ctx context.Context) error`
- **Purpose:** OnStartup implements core.Startable — starts the monitoring loop.

### Subsystem.Poke
- **File:** monitor.go
- **Signature:** `func (*Subsystem) Poke()`
- **Purpose:** Poke triggers an immediate check cycle. Prefer AgentStarted/AgentCompleted..

### Subsystem.RegisterTools
- **File:** monitor.go
- **Signature:** `func (*Subsystem) RegisterTools(server *mcp.Server)`
- **Purpose:** RegisterTools binds the monitor resource to an MCP server.

### Subsystem.SetCore
- **File:** monitor.go
- **Signature:** `func (*Subsystem) SetCore(c *core.Core)`
- **Purpose:** SetCore wires the Core framework instance and registers IPC handlers.

### Subsystem.SetNotifier
- **File:** monitor.go
- **Signature:** `func (*Subsystem) SetNotifier(n ChannelNotifier)`
- **Purpose:** SetNotifier wires up channel event broadcasting. Deprecated: Phase 3 replaces this with c.ACTION(messages.X{}).

### Subsystem.Shutdown
- **File:** monitor.go
- **Signature:** `func (*Subsystem) Shutdown(_ context.Context) error`
- **Purpose:** Shutdown stops the monitoring loop and waits for it to exit.

### Subsystem.Start
- **File:** monitor.go
- **Signature:** `func (*Subsystem) Start(ctx context.Context)`
- **Purpose:** Start begins the background monitoring loop after MCP startup.


## setup

**Import:** `dappco.re/go/agent/pkg/setup`
**Files:** 3

Package setup provides workspace setup and scaffolding using lib templates.

## Types

### Command
- **File:** config.go
- **Purpose:** Command is a named runnable command.
- **Fields:**
  - `Name string` — Name of the item.
  - `Run string` — Command line to run.

### ConfigData
- **File:** config.go
- **Purpose:** ConfigData holds the data passed to config templates.
- **Fields:**
  - `Name string` — Name of the item.
  - `Description string` — Human-readable description.
  - `Type string` — Type discriminator.
  - `Module string` — Detected Go module or project module name.
  - `Repository string` — Repository remote in owner/name form.
  - `GoVersion string` — Detected Go version.
  - `Targets []Target` — Configured build targets.
  - `Commands []Command` — Generated commands or command definitions.
  - `Env map[string]string` — Environment variables included in generated config.

### Options
- **File:** setup.go
- **Purpose:** Options controls setup behaviour.
- **Fields:**
  - `Path string` — Target directory (default: cwd)
  - `DryRun bool` — Preview only, don't write
  - `Force bool` — Overwrite existing files
  - `Template string` — Workspace template or compatibility alias (default, review, security, agent, go, php, gui, auto)

### ProjectType
- **File:** detect.go
- **Purpose:** ProjectType identifies what kind of project lives at a path.
- **Underlying Type:** `string`

### Target
- **File:** config.go
- **Purpose:** Target is a build target (os/arch pair).
- **Fields:**
  - `OS string` — Target operating system.
  - `Arch string` — Target CPU architecture.

## Functions

### Detect
- **File:** detect.go
- **Signature:** `func Detect(path string) ProjectType`
- **Purpose:** Detect identifies the project type from files present at the given path.

### DetectAll
- **File:** detect.go
- **Signature:** `func DetectAll(path string) []ProjectType`
- **Purpose:** DetectAll returns all project types found at the path (polyglot repos).

### GenerateBuildConfig
- **File:** config.go
- **Signature:** `func GenerateBuildConfig(path string, projType ProjectType) (string, error)`
- **Purpose:** GenerateBuildConfig renders a build.yaml for the detected project type.

### GenerateTestConfig
- **File:** config.go
- **Signature:** `func GenerateTestConfig(projType ProjectType) (string, error)`
- **Purpose:** GenerateTestConfig renders a test.yaml for the detected project type.

### Run
- **File:** setup.go
- **Signature:** `func Run(opts Options) error`
- **Purpose:** Run performs the workspace setup at the given path. It detects the project type, generates .core/ configs, and optionally scaffolds a workspace from a dir template.

## Methods

No exported methods.

