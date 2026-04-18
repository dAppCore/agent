<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Mod\Content\Jobs\GenerateContentJob;
use Mod\Content\Models\ContentBrief;
use Mod\Content\Services\AIGatewayService;

/**
 * Generate content for a brief using AI pipeline.
 *
 * Supports draft (Gemini), refine (Claude), or full pipeline modes.
 * Can run synchronously or queue for async processing.
 */
class ContentGenerate extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['write'];

    /**
     * Content generation can be slow, allow longer timeout.
     */
    protected ?int $timeout = 300;

    public function name(): string
    {
        return 'content_generate';
    }

    public function description(): string
    {
        return 'Generate content for a brief using AI pipeline (Gemini draft -> Claude refine)';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'brief_id' => [
                    'type' => 'integer',
                    'description' => 'Brief ID to generate content for',
                ],
                'mode' => [
                    'type' => 'string',
                    'description' => 'Generation mode',
                    'enum' => ['draft', 'refine', 'full'],
                ],
                'sync' => [
                    'type' => 'boolean',
                    'description' => 'Run synchronously (wait for result) vs queue for async processing',
                ],
            ],
            'required' => ['brief_id'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $briefId = $this->requireInt($args, 'brief_id', 1);
            $mode = $this->optionalEnum($args, 'mode', ['draft', 'refine', 'full'], 'full');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $brief = ContentBrief::find($briefId);

        if (! $brief) {
            return $this->error("Brief not found: {$briefId}");
        }

        // Optional workspace scoping
        if (! empty($context['workspace_id']) && $brief->workspace_id !== $context['workspace_id']) {
            return $this->error('Access denied: brief belongs to a different workspace');
        }

        $gateway = app(AIGatewayService::class);

        if (! $gateway->isAvailable()) {
            return $this->error('AI providers not configured. Set GOOGLE_AI_API_KEY and ANTHROPIC_API_KEY.');
        }

        $sync = $args['sync'] ?? false;

        if ($sync) {
            return $this->generateSync($brief, $gateway, $mode);
        }

        // Queue for async processing
        $brief->markQueued();
        GenerateContentJob::dispatch($brief, $mode);

        return $this->success([
            'brief_id' => $brief->id,
            'status' => 'queued',
            'mode' => $mode,
            'message' => 'Content generation queued for async processing',
        ]);
    }

    /**
     * Run generation synchronously and return results.
     */
    protected function generateSync(ContentBrief $brief, AIGatewayService $gateway, string $mode): array
    {
        try {
            if ($mode === 'full') {
                $result = $gateway->generateAndRefine($brief);

                return $this->success([
                    'brief_id' => $brief->id,
                    'status' => $brief->fresh()->status,
                    'draft' => [
                        'model' => $result['draft']->model,
                        'tokens' => $result['draft']->totalTokens(),
                        'cost' => $result['draft']->estimateCost(),
                    ],
                    'refined' => [
                        'model' => $result['refined']->model,
                        'tokens' => $result['refined']->totalTokens(),
                        'cost' => $result['refined']->estimateCost(),
                    ],
                ]);
            }

            if ($mode === 'draft') {
                $response = $gateway->generateDraft($brief);
                $brief->markDraftComplete($response->content);

                return $this->success([
                    'brief_id' => $brief->id,
                    'status' => $brief->fresh()->status,
                    'draft' => [
                        'model' => $response->model,
                        'tokens' => $response->totalTokens(),
                        'cost' => $response->estimateCost(),
                    ],
                ]);
            }

            if ($mode === 'refine') {
                if (! $brief->isGenerated()) {
                    return $this->error('No draft to refine. Generate draft first.');
                }

                $response = $gateway->refineDraft($brief, $brief->draft_output);
                $brief->markRefined($response->content);

                return $this->success([
                    'brief_id' => $brief->id,
                    'status' => $brief->fresh()->status,
                    'refined' => [
                        'model' => $response->model,
                        'tokens' => $response->totalTokens(),
                        'cost' => $response->estimateCost(),
                    ],
                ]);
            }

            return $this->error("Invalid mode: {$mode}");
        } catch (\Exception $e) {
            $brief->markFailed($e->getMessage());

            return $this->error("Generation failed: {$e->getMessage()}");
        }
    }
}
