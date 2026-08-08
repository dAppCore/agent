<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Content\Jobs\GenerateContentJob;
use Core\Mod\Content\Models\ContentBrief;

/**
 * Queue multiple briefs for batch content generation.
 *
 * Processes briefs that are ready (queued status with past or no scheduled time).
 */
class ContentBatchGenerate extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'content_batch_generate';
    }

    public function description(): string
    {
        return 'Queue multiple briefs for batch content generation';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'limit' => [
                    'type' => 'integer',
                    'description' => 'Maximum briefs to process (default: 5)',
                ],
                'mode' => [
                    'type' => 'string',
                    'description' => 'Generation mode',
                    'enum' => ['draft', 'refine', 'full'],
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $limit = $this->optionalInt($args, 'limit', 5, 1, 50);
            $mode = $this->optionalEnum($args, 'mode', ['draft', 'refine', 'full'], 'full');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $query = ContentBrief::readyToProcess();

        // Scope to workspace if provided
        if (! empty($context['workspace_id'])) {
            $query->where('workspace_id', $context['workspace_id']);
        }

        $briefs = $query->limit($limit)->get();

        if ($briefs->isEmpty()) {
            return $this->success([
                'message' => 'No briefs ready for processing',
                'queued' => 0,
            ]);
        }

        foreach ($briefs as $brief) {
            GenerateContentJob::dispatch($brief, $mode);
        }

        return $this->success([
            'queued' => $briefs->count(),
            'mode' => $mode,
            'brief_ids' => $briefs->pluck('id')->all(),
        ]);
    }
}
