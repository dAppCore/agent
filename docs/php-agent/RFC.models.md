# core/php/agent — Models

| Model | Table | Key Fields | Relationships |
|-------|-------|------------|---------------|
| `AgentPlan` | `agent_plans` | workspace_id, slug, title, description, status, agent_type, template_version_id | hasMany Phases, Sessions; belongsTo Workspace; softDeletes; logsActivity |
| `AgentPhase` | `agent_phases` | agent_plan_id, order, name, tasks (JSON), dependencies (JSON), status, completion_criteria (JSON) | belongsTo AgentPlan |
| `AgentSession` | `agent_sessions` | workspace_id, agent_plan_id, session_id (UUID), agent_type, status, context_summary (JSON), work_log (JSON), artifacts (JSON) | belongsTo Workspace, AgentPlan |
| `AgentMessage` | `agent_messages` | workspace_id, from_agent, to_agent, subject, content, read_at | belongsTo Workspace |
| `AgentApiKey` | `agent_api_keys` | workspace_id, name, key (hashed), permissions (JSON), rate_limit, call_count, last_used_at, expires_at, revoked_at | belongsTo Workspace |
| `BrainMemory` | `brain_memories` | workspace_id (UUID), agent_id, type, content, tags (JSON), project, confidence, source | belongsTo Workspace; softDeletes |
| `Issue` | `issues` | workspace_id, sprint_id, slug, title, description, type, status, priority, labels (JSON) | belongsTo Workspace, Sprint; hasMany Comments; softDeletes; logsActivity |
| `IssueComment` | `issue_comments` | issue_id, author, body, metadata (JSON) | belongsTo Issue |
| `Sprint` | `sprints` | workspace_id, slug, title, goal, status, metadata (JSON), started_at, ended_at | belongsTo Workspace; hasMany Issues; softDeletes; logsActivity |
| `Task` | `tasks` | workspace_id, title, description, status, priority, category, file_ref, line_ref | belongsTo Workspace |
| `Prompt` | `prompts` | name, category, description, system_prompt, user_template, variables (JSON), model, model_config (JSON), is_active | hasMany Versions, ContentTasks |
| `PromptVersion` | `prompt_versions` | prompt_id, version, system_prompt, user_template, variables (JSON), created_by | belongsTo Prompt, User |
| `PlanTemplateVersion` | `plan_template_versions` | slug, version, name, content (JSON), content_hash (SHA-256) | hasMany AgentPlans |
| `WorkspaceState` | `workspace_states` | agent_plan_id, key, value (JSON), type, description | belongsTo AgentPlan |
| `FleetNode` | `fleet_nodes` | workspace_id, agent_id (unique), platform, models (JSON), capabilities (JSON), status, compute_budget (JSON: {max_daily_hours, max_weekly_cost_usd, quiet_start, quiet_end, prefer_models[], avoid_models[]}), current_task_id (nullable FK), last_heartbeat_at, registered_at | belongsTo Workspace; belongsTo FleetTask (current) |
| `FleetTask` | `fleet_tasks` | workspace_id, fleet_node_id, repo, branch, task, template, agent_model, status, result (JSON), findings (JSON), changes (JSON: files_changed, insertions, deletions), report (JSON), started_at, completed_at | belongsTo Workspace, FleetNode |
| `CreditEntry` | `credit_entries` | workspace_id, fleet_node_id, task_type, amount, balance_after, description | belongsTo Workspace, FleetNode |
| `SyncRecord` | `sync_records` | fleet_node_id, direction (push/pull), payload_size, items_count, synced_at | belongsTo FleetNode |
