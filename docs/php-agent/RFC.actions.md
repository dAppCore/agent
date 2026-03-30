# core/php/agent — Actions

## Brain
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Remember | `Actions\Brain\RememberKnowledge` | content, tags[], project? | BrainMemory |
| Recall | `Actions\Brain\RecallKnowledge` | query, limit?, tags[]? | BrainMemory[] |
| Forget | `Actions\Brain\ForgetKnowledge` | id | bool |
| List | `Actions\Brain\ListKnowledge` | filters? | BrainMemory[] |

## Forge
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Assign Agent | `Actions\Forge\AssignAgent` | issue_id, agent_type | bool |
| Create Plan from Issue | `Actions\Forge\CreatePlanFromIssue` | issue_id | AgentPlan |
| Manage PR | `Actions\Forge\ManagePullRequest` | pr_id, action | bool |
| Report to Issue | `Actions\Forge\ReportToIssue` | issue_id, report | bool |
| Scan for Work | `Actions\Forge\ScanForWork` | — | Issue[] |

## Plan
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Create | `Actions\Plan\CreatePlan` | title, description, phases[] | AgentPlan |
| Get | `Actions\Plan\GetPlan` | id or slug | AgentPlan |
| List | `Actions\Plan\ListPlans` | status?, workspace_id? | AgentPlan[] |
| Update Status | `Actions\Plan\UpdatePlanStatus` | id, status | AgentPlan |
| Archive | `Actions\Plan\ArchivePlan` | id | bool |

## Session
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Start | `Actions\Session\StartSession` | agent_type, plan_id?, context | AgentSession |
| Continue | `Actions\Session\ContinueSession` | session_id, work_log | AgentSession |
| End | `Actions\Session\EndSession` | session_id, summary, handoff? | AgentSession |
| Get | `Actions\Session\GetSession` | session_id | AgentSession |
| List | `Actions\Session\ListSessions` | status?, agent_type? | AgentSession[] |

## Issue
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Create | `Actions\Issue\CreateIssue` | title, type, priority, labels[] | Issue |
| Get | `Actions\Issue\GetIssue` | id | Issue |
| List | `Actions\Issue\ListIssues` | status?, type?, sprint_id? | Issue[] |
| Update | `Actions\Issue\UpdateIssue` | id, fields | Issue |
| Comment | `Actions\Issue\AddIssueComment` | issue_id, body | IssueComment |
| Archive | `Actions\Issue\ArchiveIssue` | id | bool |

## Sprint
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Create | `Actions\Sprint\CreateSprint` | title, goal, started_at, ended_at | Sprint |
| Get | `Actions\Sprint\GetSprint` | id | Sprint |
| List | `Actions\Sprint\ListSprints` | status? | Sprint[] |
| Update | `Actions\Sprint\UpdateSprint` | id, fields | Sprint |
| Archive | `Actions\Sprint\ArchiveSprint` | id | bool |

## Phase
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Get | `Actions\Phase\GetPhase` | id | AgentPhase |
| Update Status | `Actions\Phase\UpdatePhaseStatus` | id, status | AgentPhase |
| Add Checkpoint | `Actions\Phase\AddCheckpoint` | id, checkpoint_data | AgentPhase |

## Task
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Toggle | `Actions\Task\ToggleTask` | id | Task |
| Update | `Actions\Task\UpdateTask` | id, fields | Task |

## Auth
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| ProvisionKey | `Actions\Auth\ProvisionAgentKey` | oauth_user_id, name?, permissions[]? | AgentApiKey |
| RevokeKey | `Actions\Auth\RevokeAgentKey` | key_id | bool |

## Fleet
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| Register | `Actions\Fleet\RegisterNode` | agent_id, capabilities, platform, models[] | FleetNode |
| Heartbeat | `Actions\Fleet\NodeHeartbeat` | agent_id, status, compute_budget | FleetNode |
| Deregister | `Actions\Fleet\DeregisterNode` | agent_id | bool |
| ListNodes | `Actions\Fleet\ListNodes` | status?, platform? | FleetNode[] |
| AssignTask | `Actions\Fleet\AssignTask` | agent_id, task, repo, template | FleetTask |
| CompleteTask | `Actions\Fleet\CompleteTask` | agent_id, task_id, result, findings[] | FleetTask (triggers AwardCredits as side-effect) |
| GetNextTask | `Actions\Fleet\GetNextTask` | agent_id, capabilities | FleetTask? (scheduler: P0-P3 priority, capability match, repo affinity, round-robin, budget check) |

## Fleet Stats
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| GetFleetStats | `Actions\Fleet\GetFleetStats` | (none) | FleetStats |

## Sync
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| PushState | `Actions\Sync\PushDispatchHistory` | agent_id, dispatches[] | SyncResult |
| PullContext | `Actions\Sync\PullFleetContext` | agent_id, since? | FleetContext |
| GetStatus | `Actions\Sync\GetAgentSyncStatus` | agent_id | SyncStatus |

## Credits
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| AwardCredits | `Actions\Credits\AwardCredits` | agent_id, task_type, amount | CreditEntry |
| GetBalance | `Actions\Credits\GetBalance` | agent_id | CreditBalance |
| GetHistory | `Actions\Credits\GetCreditHistory` | agent_id, limit? | CreditEntry[] |

## Subscription
| Action | Class | Input | Output |
|--------|-------|-------|--------|
| DetectCapabilities | `Actions\Subscription\DetectCapabilities` | api_keys{} | Capabilities |
| GetNodeBudget | `Actions\Subscription\GetNodeBudget` | agent_id | Budget |
| UpdateBudget | `Actions\Subscription\UpdateBudget` | agent_id, limits | Budget |
