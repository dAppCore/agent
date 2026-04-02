<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Mod\Agentic\Database\Factories\AgentSessionFactory;
use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Ramsey\Uuid\Uuid;

/**
 * Agent Session - tracks agent work sessions for handoff.
 *
 * Enables context recovery and multi-agent collaboration
 * by logging actions, artifacts, and handoff notes.
 *
 * @property int $id
 * @property int $workspace_id
 * @property int|null $agent_plan_id
 * @property string $session_id
 * @property string|null $agent_type
 * @property string $status
 * @property array|null $context_summary
 * @property array|null $work_log
 * @property array|null $artifacts
 * @property array|null $handoff_notes
 * @property string|null $final_summary
 * @property \Carbon\Carbon $started_at
 * @property \Carbon\Carbon $last_active_at
 * @property \Carbon\Carbon|null $ended_at
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class AgentSession extends Model
{
    use BelongsToWorkspace;

    /** @use HasFactory<AgentSessionFactory> */
    use HasFactory;

    protected static function newFactory(): AgentSessionFactory
    {
        return AgentSessionFactory::new();
    }

    protected $fillable = [
        'workspace_id',
        'agent_api_key_id',
        'agent_plan_id',
        'session_id',
        'agent_type',
        'status',
        'context_summary',
        'work_log',
        'artifacts',
        'handoff_notes',
        'final_summary',
        'started_at',
        'last_active_at',
        'ended_at',
    ];

    protected $casts = [
        'context_summary' => 'array',
        'work_log' => 'array',
        'artifacts' => 'array',
        'handoff_notes' => 'array',
        'started_at' => 'datetime',
        'last_active_at' => 'datetime',
        'ended_at' => 'datetime',
    ];

    // Status constants
    public const STATUS_ACTIVE = 'active';

    public const STATUS_PAUSED = 'paused';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_FAILED = 'failed';

    public const STATUS_HANDED_OFF = 'handed_off';

    // Agent types
    public const AGENT_OPUS = 'opus';

    public const AGENT_SONNET = 'sonnet';

    public const AGENT_HAIKU = 'haiku';

    // Relationships
    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function plan(): BelongsTo
    {
        return $this->belongsTo(AgentPlan::class, 'agent_plan_id');
    }

    public function apiKey(): BelongsTo
    {
        return $this->belongsTo(AgentApiKey::class, 'agent_api_key_id');
    }

    // Scopes
    public function scopeActive(Builder $query): Builder
    {
        return $query->where('status', self::STATUS_ACTIVE);
    }

    public function scopeForPlan(Builder $query, AgentPlan|int $plan): Builder
    {
        $planId = $plan instanceof AgentPlan ? $plan->id : $plan;

        return $query->where('agent_plan_id', $planId);
    }

    // Factory
    public static function start(?AgentPlan $plan = null, ?string $agentType = null, ?Workspace $workspace = null): self
    {
        $workspaceId = $workspace?->id ?? $plan?->workspace_id;

        return static::create([
            'workspace_id' => $workspaceId,
            'agent_plan_id' => $plan?->id,
            'session_id' => 'sess_'.Uuid::uuid4()->toString(),
            'agent_type' => $agentType,
            'status' => self::STATUS_ACTIVE,
            'work_log' => [],
            'artifacts' => [],
            'started_at' => now(),
            'last_active_at' => now(),
        ]);
    }

    // Status helpers
    public function isActive(): bool
    {
        return $this->status === self::STATUS_ACTIVE;
    }

    public function isPaused(): bool
    {
        return $this->status === self::STATUS_PAUSED;
    }

    public function isEnded(): bool
    {
        return in_array($this->status, [self::STATUS_COMPLETED, self::STATUS_FAILED, self::STATUS_HANDED_OFF], true);
    }

    // Actions
    public function touchActivity(): self
    {
        $this->update(['last_active_at' => now()]);

        return $this;
    }

    public function pause(): self
    {
        $this->update(['status' => self::STATUS_PAUSED]);

        return $this;
    }

    public function resume(): self
    {
        $this->update([
            'status' => self::STATUS_ACTIVE,
            'last_active_at' => now(),
        ]);

        return $this;
    }

    public function complete(?string $summary = null): self
    {
        $this->update([
            'status' => self::STATUS_COMPLETED,
            'final_summary' => $summary,
            'ended_at' => now(),
        ]);

        return $this;
    }

    public function fail(?string $reason = null): self
    {
        $this->update([
            'status' => self::STATUS_FAILED,
            'final_summary' => $reason,
            'ended_at' => now(),
        ]);

        return $this;
    }

    // Work log
    public function logAction(string $action, ?array $details = null): self
    {
        $log = $this->work_log ?? [];
        $log[] = [
            'action' => $action,
            'details' => $details,
            'timestamp' => now()->toIso8601String(),
        ];
        $this->update([
            'work_log' => $log,
            'last_active_at' => now(),
        ]);

        return $this;
    }

    /**
     * Add a typed work log entry.
     */
    public function addWorkLogEntry(string $message, string $type = 'info', array $data = []): self
    {
        $log = $this->work_log ?? [];
        $log[] = [
            'message' => $message,
            'type' => $type, // info, warning, error, success, checkpoint
            'data' => $data,
            'timestamp' => now()->toIso8601String(),
        ];
        $this->update([
            'work_log' => $log,
            'last_active_at' => now(),
        ]);

        return $this;
    }

    /**
     * End the session with a status.
     */
    public function end(string $status, ?string $summary = null, ?array $handoffNotes = null): self
    {
        $validStatuses = [self::STATUS_COMPLETED, self::STATUS_FAILED, self::STATUS_HANDED_OFF];

        if (! in_array($status, $validStatuses, true)) {
            $status = self::STATUS_COMPLETED;
        }

        $this->update([
            'status' => $status,
            'final_summary' => $summary,
            'handoff_notes' => $handoffNotes ?? $this->handoff_notes,
            'ended_at' => now(),
        ]);

        return $this;
    }

    public function getRecentActions(int $limit = 10): array
    {
        $log = $this->work_log ?? [];

        return array_slice(array_reverse($log), 0, $limit);
    }

    // Artifacts
    public function addArtifact(string $path, string $action = 'modified', ?array $metadata = null): self
    {
        $artifacts = $this->artifacts ?? [];
        $artifacts[] = [
            'path' => $path,
            'action' => $action, // created, modified, deleted
            'metadata' => $metadata,
            'timestamp' => now()->toIso8601String(),
        ];
        $this->update(['artifacts' => $artifacts]);

        return $this;
    }

    public function getArtifactsByAction(string $action): array
    {
        $artifacts = $this->artifacts ?? [];

        return array_filter($artifacts, fn ($a) => ($a['action'] ?? '') === $action);
    }

    // Context summary
    public function updateContextSummary(array $summary): self
    {
        $this->update(['context_summary' => $summary]);

        return $this;
    }

    public function addToContext(string $key, mixed $value): self
    {
        $context = $this->context_summary ?? [];
        $context[$key] = $value;
        $this->update(['context_summary' => $context]);

        return $this;
    }

    // Handoff
    public function prepareHandoff(
        string $summary,
        array $nextSteps = [],
        array $blockers = [],
        array $contextForNext = []
    ): self {
        $this->update([
            'handoff_notes' => [
                'summary' => $summary,
                'next_steps' => $nextSteps,
                'blockers' => $blockers,
                'context_for_next' => $contextForNext,
            ],
            'status' => self::STATUS_PAUSED,
        ]);

        return $this;
    }

    public function getHandoffContext(): array
    {
        $context = [
            'session_id' => $this->session_id,
            'agent_type' => $this->agent_type,
            'started_at' => $this->started_at?->toIso8601String(),
            'last_active_at' => $this->last_active_at?->toIso8601String(),
            'context_summary' => $this->context_summary,
            'recent_actions' => $this->getRecentActions(20),
            'artifacts' => $this->artifacts,
            'handoff_notes' => $this->handoff_notes,
        ];

        if ($this->plan) {
            $context['plan'] = [
                'slug' => $this->plan->slug,
                'title' => $this->plan->title,
                'current_phase' => $this->plan->current_phase,
                'progress' => $this->plan->getProgress(),
            ];
        }

        return $context;
    }

    // Replay functionality

    /**
     * Get the replay context - reconstructs session state from work log.
     *
     * This provides the data needed to resume/replay a session by analysing
     * the work log entries to understand what was done and what state the
     * session was in.
     */
    public function getReplayContext(): array
    {
        $workLog = $this->work_log ?? [];
        $artifacts = $this->artifacts ?? [];

        // Extract checkpoints from work log
        $checkpoints = array_values(array_filter(
            $workLog,
            fn ($entry) => ($entry['type'] ?? '') === 'checkpoint'
        ));

        // Get the last checkpoint if any
        $lastCheckpoint = ! empty($checkpoints) ? end($checkpoints) : null;

        // Extract decisions made during the session
        $decisions = array_values(array_filter(
            $workLog,
            fn ($entry) => ($entry['type'] ?? '') === 'decision'
        ));

        // Extract errors encountered
        $errors = array_values(array_filter(
            $workLog,
            fn ($entry) => ($entry['type'] ?? '') === 'error'
        ));

        // Build a progress summary from the work log
        $progressSummary = $this->buildProgressSummary($workLog);

        return [
            'session_id' => $this->session_id,
            'status' => $this->status,
            'agent_type' => $this->agent_type,
            'plan' => $this->plan ? [
                'slug' => $this->plan->slug,
                'title' => $this->plan->title,
                'current_phase' => $this->plan->current_phase,
            ] : null,
            'started_at' => $this->started_at?->toIso8601String(),
            'last_active_at' => $this->last_active_at?->toIso8601String(),
            'duration' => $this->getDurationFormatted(),

            // Reconstructed state
            'context_summary' => $this->context_summary,
            'progress_summary' => $progressSummary,
            'last_checkpoint' => $lastCheckpoint,
            'checkpoints' => $checkpoints,
            'decisions' => $decisions,
            'errors' => $errors,

            // Artifacts created during session
            'artifacts' => $artifacts,
            'artifacts_by_action' => [
                'created' => $this->getArtifactsByAction('created'),
                'modified' => $this->getArtifactsByAction('modified'),
                'deleted' => $this->getArtifactsByAction('deleted'),
            ],

            // Recent work for context
            'recent_actions' => $this->getRecentActions(20),
            'total_actions' => count($workLog),

            // Handoff state if available
            'handoff_notes' => $this->handoff_notes,
            'final_summary' => $this->final_summary,
        ];
    }

    /**
     * Build a progress summary from work log entries.
     */
    protected function buildProgressSummary(array $workLog): array
    {
        if (empty($workLog)) {
            return [
                'completed_steps' => 0,
                'last_action' => null,
                'summary' => 'No work recorded',
            ];
        }

        $lastEntry = end($workLog);
        $checkpointCount = count(array_filter($workLog, fn ($e) => ($e['type'] ?? '') === 'checkpoint'));
        $errorCount = count(array_filter($workLog, fn ($e) => ($e['type'] ?? '') === 'error'));

        return [
            'completed_steps' => count($workLog),
            'checkpoint_count' => $checkpointCount,
            'error_count' => $errorCount,
            'last_action' => $lastEntry['action'] ?? $lastEntry['message'] ?? 'Unknown',
            'last_action_at' => $lastEntry['timestamp'] ?? null,
            'summary' => sprintf(
                '%d actions recorded, %d checkpoints, %d errors',
                count($workLog),
                $checkpointCount,
                $errorCount
            ),
        ];
    }

    /**
     * Create a new session that continues from this one (replay).
     *
     * This creates a fresh session with the context from this session,
     * allowing an agent to pick up where this session left off.
     */
    public function createReplaySession(?string $agentType = null): self
    {
        $replayContext = $this->getReplayContext();

        $newSession = static::create([
            'workspace_id' => $this->workspace_id,
            'agent_plan_id' => $this->agent_plan_id,
            'session_id' => 'ses_replay_'.now()->format('Ymd_His').'_'.substr(md5((string) $this->id), 0, 8),
            'agent_type' => $agentType ?? $this->agent_type,
            'status' => self::STATUS_ACTIVE,
            'started_at' => now(),
            'last_active_at' => now(),
            'context_summary' => [
                'replayed_from' => $this->session_id,
                'original_started_at' => $this->started_at?->toIso8601String(),
                'original_status' => $this->status,
                'inherited_context' => $this->context_summary,
                'replay_checkpoint' => $replayContext['last_checkpoint'],
                'original_progress' => $replayContext['progress_summary'],
            ],
            'work_log' => [
                [
                    'message' => sprintf('Replayed from session %s', $this->session_id),
                    'type' => 'info',
                    'data' => [
                        'original_session' => $this->session_id,
                        'original_actions' => $replayContext['total_actions'],
                        'original_checkpoints' => count($replayContext['checkpoints']),
                    ],
                    'timestamp' => now()->toIso8601String(),
                ],
            ],
            'artifacts' => [],
            'handoff_notes' => $this->handoff_notes,
        ]);

        return $newSession;
    }

    // Duration helpers
    public function getDuration(): ?int
    {
        if (! $this->started_at) {
            return null;
        }

        $end = $this->ended_at ?? now();

        return (int) $this->started_at->diffInMinutes($end);
    }

    public function getDurationFormatted(): string
    {
        $minutes = $this->getDuration();
        if ($minutes === null) {
            return 'Unknown';
        }

        if ($minutes < 60) {
            return "{$minutes}m";
        }

        $hours = floor($minutes / 60);
        $mins = $minutes % 60;

        return "{$hours}h {$mins}m";
    }

    // Output
    public function toMcpContext(): array
    {
        return [
            'session_id' => $this->session_id,
            'agent_type' => $this->agent_type,
            'status' => $this->status,
            'workspace_id' => $this->workspace_id,
            'plan_slug' => $this->plan?->slug,
            'started_at' => $this->started_at?->toIso8601String(),
            'last_active_at' => $this->last_active_at?->toIso8601String(),
            'ended_at' => $this->ended_at?->toIso8601String(),
            'duration' => $this->getDurationFormatted(),
            'action_count' => count($this->work_log ?? []),
            'artifact_count' => count($this->artifacts ?? []),
            'context_summary' => $this->context_summary,
            'handoff_notes' => $this->handoff_notes,
        ];
    }
}
