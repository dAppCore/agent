# Issue Tracker Implementation Plan

> **For agentic workers:** Follow this plan phase by phase. Commit after each phase.

**Goal:** Add Issue, Sprint, and IssueComment models to the php-agentic module with migrations, API endpoints, and Actions.

**Location:** `/Users/snider/Code/core/agent/src/php/`
**Spec:** `/Users/snider/Code/host-uk/specs/RFC-024-ISSUE-TRACKER.md`

---

## Phase 1: Migration

Create migration file: `src/php/Migrations/0001_01_01_000010_create_issue_tracker_tables.php`

Three tables: `issues`, `sprints`, `issue_comments`

Issues table: id, workspace_id (FK), repo (string), title (string), body (text nullable), status (string default 'open'), priority (string default 'normal'), milestone (string default 'backlog'), size (string default 'small'), source (string nullable), source_ref (string nullable), assignee (string nullable), labels (json nullable), pr_url (string nullable), plan_id (FK nullable to agent_plans), parent_id (FK nullable self-referencing), metadata (json nullable), timestamps, soft deletes. Indexes on (workspace_id, status), (workspace_id, milestone), (workspace_id, repo), parent_id.

Sprints table: id, workspace_id (FK), name (string), status (string default 'planning'), started_at (timestamp nullable), completed_at (timestamp nullable), notes (text nullable), metadata (json nullable), timestamps.

Issue comments table: id, issue_id (FK cascade delete), author (string), body (text), type (string default 'comment'), metadata (json nullable), timestamps.

Use hasTable() guards for idempotency like existing migrations.

**Commit: feat(tracker): add issue tracker migrations**

## Phase 2: Models

Create three models following existing patterns (BelongsToWorkspace trait, strict types, UK English):

`src/php/Models/Issue.php`:
- Fillable: repo, title, body, status, priority, milestone, size, source, source_ref, assignee, labels, pr_url, plan_id, parent_id, metadata
- Casts: labels as array, metadata as array
- Status constants: STATUS_OPEN, STATUS_ASSIGNED, STATUS_IN_PROGRESS, STATUS_REVIEW, STATUS_DONE, STATUS_CLOSED
- Priority constants: PRIORITY_CRITICAL, PRIORITY_HIGH, PRIORITY_NORMAL, PRIORITY_LOW
- Milestone constants: MILESTONE_NEXT_PATCH, MILESTONE_NEXT_MINOR, MILESTONE_NEXT_MAJOR, MILESTONE_IDEAS, MILESTONE_BACKLOG
- Size constants: SIZE_TRIVIAL, SIZE_SMALL, SIZE_MEDIUM, SIZE_LARGE, SIZE_EPIC
- Relations: plan() belongsTo AgentPlan, parent() belongsTo Issue, children() hasMany Issue, comments() hasMany IssueComment
- Scopes: scopeOpen, scopeByRepo, scopeByMilestone, scopeByPriority, scopeEpics (where parent_id is null and size is epic)
- Methods: isEpic(), assign(string), markInProgress(), markReview(string prUrl), markDone(), close()
- Use SoftDeletes, LogsActivity (title, status)

`src/php/Models/Sprint.php`:
- Fillable: name, status, started_at, completed_at, notes, metadata
- Casts: started_at as datetime, completed_at as datetime, metadata as array
- Status constants: STATUS_PLANNING, STATUS_ACTIVE, STATUS_COMPLETED
- Methods: start(), complete()
- start(): sets status to active, started_at to now(). Updates all issues in next-* milestones to status assigned.
- complete(): sets status to completed, completed_at to now().

`src/php/Models/IssueComment.php`:
- Fillable: issue_id, author, body, type, metadata
- Casts: metadata as array
- Type constants: TYPE_COMMENT, TYPE_TRIAGE, TYPE_SCAN_RESULT, TYPE_STATUS_CHANGE
- Relations: issue() belongsTo Issue

**Commit: feat(tracker): add Issue, Sprint, IssueComment models**

## Phase 3: API Controller + Routes

Create `src/php/Controllers/Api/IssueController.php`:
- index: list issues with filters (repo, status, milestone, priority, assignee). Paginated.
- show: get issue with comments and children count
- store: create issue with validation
- update: patch issue fields
- destroy: soft delete

Create `src/php/Controllers/Api/SprintController.php`:
- index: list sprints
- store: create sprint
- start: POST /sprints/{id}/start
- complete: POST /sprints/{id}/complete

Add routes to `src/php/Routes/api.php`:
```
Route::apiResource('issues', IssueController::class);
Route::post('issues/{issue}/comments', [IssueController::class, 'addComment']);
Route::get('issues/{issue}/comments', [IssueController::class, 'listComments']);
Route::apiResource('sprints', SprintController::class)->only(['index', 'store']);
Route::post('sprints/{sprint}/start', [SprintController::class, 'start']);
Route::post('sprints/{sprint}/complete', [SprintController::class, 'complete']);
```

All protected by AgentApiAuth middleware.

**Commit: feat(tracker): add issue and sprint API endpoints**

## Phase 4: Actions

Create `src/php/Actions/Issue/CreateIssueFromScan.php`:
- Takes scan results (repo, findings array, source type)
- Creates one issue per finding or one issue with findings in body
- Sets source, source_ref, labels from scan type
- Sets milestone based on priority (critical/high -> next-patch, normal -> next-minor, low -> backlog)

Create `src/php/Actions/Issue/TriageIssue.php`:
- Takes issue and triage data (size, priority, milestone, notes)
- Updates issue fields
- Adds triage comment with author and notes

Create `src/php/Actions/Sprint/CompleteSprint.php`:
- Gets all done issues grouped by repo
- Generates changelog per repo
- Stores changelog in sprint metadata
- Closes done issues

**Commit: feat(tracker): add issue and sprint actions**
