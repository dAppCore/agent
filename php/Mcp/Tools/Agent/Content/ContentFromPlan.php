<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Content\Enums\BriefContentType;
use Core\Mod\Content\Jobs\GenerateContentJob;
use Core\Mod\Content\Models\ContentBrief;
use Illuminate\Support\Str;

/**
 * Create content briefs from plan tasks and queue for generation.
 *
 * Converts pending tasks from a plan into content briefs, enabling
 * automated content generation workflows from plan-based task management.
 */
class ContentFromPlan extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'content_from_plan';
    }

    public function description(): string
    {
        return 'Create content briefs from plan tasks and queue for generation';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'plan_slug' => [
                    'type' => 'string',
                    'description' => 'Plan slug to generate content from',
                ],
                'content_type' => [
                    'type' => 'string',
                    'description' => 'Type of content to generate',
                    'enum' => BriefContentType::values(),
                ],
                'service' => [
                    'type' => 'string',
                    'description' => 'Service context',
                ],
                'limit' => [
                    'type' => 'integer',
                    'description' => 'Maximum briefs to create (default: 5)',
                ],
                'target_word_count' => [
                    'type' => 'integer',
                    'description' => 'Target word count per article',
                ],
            ],
            'required' => ['plan_slug'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $planSlug = $this->requireString($args, 'plan_slug', 255);
            $limit = $this->optionalInt($args, 'limit', 5, 1, 50);
            $wordCount = $this->optionalInt($args, 'target_word_count', 800, 100, 10000);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $plan = AgentPlan::with('agentPhases')
            ->where('slug', $planSlug)
            ->first();

        if (! $plan) {
            return $this->error("Plan not found: {$planSlug}");
        }

        $contentType = $args['content_type'] ?? 'help_article';
        $service = $args['service'] ?? ($plan->context['service'] ?? null);

        // Get workspace_id from context
        $workspaceId = $context['workspace_id'] ?? $plan->workspace_id;

        $phases = $plan->agentPhases()
            ->whereIn('status', ['pending', 'in_progress'])
            ->get();

        if ($phases->isEmpty()) {
            return $this->success([
                'message' => 'No pending phases in plan',
                'created' => 0,
            ]);
        }

        $briefsCreated = [];

        foreach ($phases as $phase) {
            $tasks = $phase->tasks ?? [];

            foreach ($tasks as $index => $task) {
                if (count($briefsCreated) >= $limit) {
                    break 2;
                }

                $taskName = is_string($task) ? $task : ($task['name'] ?? '');
                $taskStatus = is_array($task) ? ($task['status'] ?? 'pending') : 'pending';

                // Skip completed tasks
                if ($taskStatus === 'completed' || empty($taskName)) {
                    continue;
                }

                // Create brief from task
                $brief = ContentBrief::create([
                    'workspace_id' => $workspaceId,
                    'title' => $taskName,
                    'slug' => Str::slug($taskName).'-'.Str::random(6),
                    'content_type' => $contentType,
                    'service' => $service,
                    'target_word_count' => $wordCount,
                    'status' => ContentBrief::STATUS_QUEUED,
                    'metadata' => [
                        'plan_id' => $plan->id,
                        'plan_slug' => $plan->slug,
                        'phase_order' => $phase->order,
                        'phase_name' => $phase->name,
                        'task_index' => $index,
                    ],
                ]);

                // Queue for generation
                GenerateContentJob::dispatch($brief, 'full');

                $briefsCreated[] = [
                    'id' => $brief->id,
                    'title' => $brief->title,
                    'phase' => $phase->name,
                ];
            }
        }

        if (empty($briefsCreated)) {
            return $this->success([
                'message' => 'No eligible tasks found (all completed or empty)',
                'created' => 0,
            ]);
        }

        return $this->success([
            'created' => count($briefsCreated),
            'content_type' => $contentType,
            'service' => $service,
            'briefs' => $briefsCreated,
        ]);
    }
}
