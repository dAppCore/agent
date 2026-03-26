<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Template;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Services\PlanTemplateService;

/**
 * Create a new plan from a template.
 */
class TemplateCreatePlan extends AgentTool
{
    protected string $category = 'template';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'template_create_plan';
    }

    public function description(): string
    {
        return 'Create a new plan from a template';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'template' => [
                    'type' => 'string',
                    'description' => 'Template name/slug',
                ],
                'variables' => [
                    'type' => 'object',
                    'description' => 'Variable values for the template',
                ],
                'slug' => [
                    'type' => 'string',
                    'description' => 'Custom slug for the plan',
                ],
            ],
            'required' => ['template', 'variables'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $templateSlug = $this->require($args, 'template');
            $variables = $this->require($args, 'variables');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $templateService = app(PlanTemplateService::class);

        $options = [];
        $customSlug = $this->optional($args, 'slug');
        if (! empty($customSlug)) {
            $options['slug'] = $customSlug;
        }

        if (isset($context['workspace_id'])) {
            $options['workspace_id'] = $context['workspace_id'];
        }

        try {
            $plan = $templateService->createPlan($templateSlug, $variables, $options);
        } catch (\Throwable $e) {
            return $this->error('Failed to create plan from template: '.$e->getMessage());
        }

        if (! $plan) {
            return $this->error('Failed to create plan from template');
        }

        $phases = $plan->agentPhases;
        $progress = $plan->getProgress();

        return $this->success([
            'plan' => [
                'slug' => $plan->slug,
                'title' => $plan->title,
                'status' => $plan->status,
                'phases' => $phases?->count() ?? 0,
                'total_tasks' => $progress['total'] ?? 0,
            ],
            'commands' => [
                'view' => "php artisan plan:show {$plan->slug}",
                'activate' => "php artisan plan:status {$plan->slug} --set=active",
            ],
        ]);
    }
}
