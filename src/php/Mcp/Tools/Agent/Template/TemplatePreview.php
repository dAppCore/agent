<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Template;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Services\PlanTemplateService;

/**
 * Preview a template with variables.
 */
class TemplatePreview extends AgentTool
{
    protected string $category = 'template';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'template_preview';
    }

    public function description(): string
    {
        return 'Preview a template with variables';
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
            ],
            'required' => ['template'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $templateSlug = $this->require($args, 'template');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $templateService = app(PlanTemplateService::class);
        $variables = $this->optional($args, 'variables', []);

        $preview = $templateService->previewTemplate($templateSlug, $variables);

        if (! $preview) {
            return $this->error("Template not found: {$templateSlug}");
        }

        return [
            'template' => $templateSlug,
            'preview' => $preview,
        ];
    }
}
