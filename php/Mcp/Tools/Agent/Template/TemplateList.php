<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Template;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Services\PlanTemplateService;

/**
 * List available plan templates.
 */
class TemplateList extends AgentTool
{
    protected string $category = 'template';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'template_list';
    }

    public function description(): string
    {
        return 'List available plan templates';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'category' => [
                    'type' => 'string',
                    'description' => 'Filter by category',
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $templateService = app(PlanTemplateService::class);
        $templates = $templateService->listTemplates();

        $category = $this->optional($args, 'category');
        if (! empty($category)) {
            $templates = array_filter($templates, fn ($t) => ($t['category'] ?? '') === $category);
        }

        return [
            'templates' => array_values($templates),
            'total' => count($templates),
        ];
    }
}
