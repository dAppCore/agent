<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Jobs;

use Core\Mod\Agentic\Services\AgenticManager;
use Core\Tenant\Services\EntitlementService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Mod\Content\Models\ContentTask;
use Throwable;

class ProcessContentTask implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $tries = 3;

    public int $backoff = 60;

    public int $timeout = 300;

    public function __construct(
        public ContentTask $task
    ) {
        $this->onQueue('ai');
    }

    public function handle(
        AgenticManager $ai,
        EntitlementService $entitlements
    ): void {
        $this->task->markProcessing();

        $prompt = $this->task->prompt;

        if (! $prompt) {
            $this->task->markFailed('Prompt not found');

            return;
        }

        // Check AI credits entitlement
        $workspace = $this->task->workspace;

        if ($workspace) {
            $result = $entitlements->can($workspace, 'ai.credits');

            if ($result->isDenied()) {
                $this->task->markFailed("Entitlement denied: {$result->message}");

                return;
            }
        }

        $provider = $ai->provider($prompt->model);

        if (! $provider->isAvailable()) {
            $this->task->markFailed("AI provider '{$prompt->model}' is not configured");

            return;
        }

        // Interpolate variables into the user template
        $userPrompt = $this->interpolateVariables(
            $prompt->user_template,
            $this->task->input_data ?? []
        );

        $response = $provider->generate(
            $prompt->system_prompt,
            $userPrompt,
            $prompt->model_config ?? []
        );

        $this->task->markCompleted($response->content, [
            'tokens_input' => $response->inputTokens,
            'tokens_output' => $response->outputTokens,
            'model' => $response->model,
            'duration_ms' => $response->durationMs,
            'estimated_cost' => $response->estimateCost(),
        ]);

        // Record AI usage
        if ($workspace) {
            $entitlements->recordUsage(
                $workspace,
                'ai.credits',
                quantity: 1,
                metadata: [
                    'task_id' => $this->task->id,
                    'prompt_id' => $prompt->id,
                    'model' => $response->model,
                    'tokens_input' => $response->inputTokens,
                    'tokens_output' => $response->outputTokens,
                    'estimated_cost' => $response->estimateCost(),
                ]
            );
        }
    }

    public function failed(Throwable $exception): void
    {
        $this->task->markFailed($exception->getMessage());
    }

    private function interpolateVariables(string $template, array $data): string
    {
        foreach ($data as $key => $value) {
            $placeholder = '{{{'.$key.'}}}';

            if (is_string($value)) {
                $template = str_replace($placeholder, $value, $template);
            } elseif (is_array($value)) {
                $template = str_replace($placeholder, json_encode($value), $template);
            }
        }

        return $template;
    }
}
