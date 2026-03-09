<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\PlanTemplateVersion;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Str;
use Symfony\Component\Yaml\Yaml;

/**
 * Plan Template Service - creates plans from YAML templates.
 *
 * Templates define reusable plan structures with phases, tasks,
 * and variable substitution for customisation.
 */
class PlanTemplateService
{
    protected string $templatesPath;

    public function __construct()
    {
        $this->templatesPath = resource_path('plan-templates');
    }

    /**
     * List all available templates.
     */
    public function list(): Collection
    {
        if (! File::isDirectory($this->templatesPath)) {
            return collect();
        }

        return collect(File::files($this->templatesPath))
            ->filter(fn ($file) => $file->getExtension() === 'yaml' || $file->getExtension() === 'yml')
            ->map(function ($file) {
                $content = Yaml::parseFile($file->getPathname());

                // Transform variables from keyed dict to indexed array for display
                $variables = collect($content['variables'] ?? [])
                    ->map(fn ($config, $name) => [
                        'name' => $name,
                        'description' => $config['description'] ?? null,
                        'default' => $config['default'] ?? null,
                        'required' => $config['required'] ?? false,
                    ])
                    ->values()
                    ->toArray();

                return [
                    'slug' => pathinfo($file->getFilename(), PATHINFO_FILENAME),
                    'name' => $content['name'] ?? Str::title(pathinfo($file->getFilename(), PATHINFO_FILENAME)),
                    'description' => $content['description'] ?? null,
                    'category' => $content['category'] ?? 'general',
                    'phases_count' => count($content['phases'] ?? []),
                    'variables' => $variables,
                    'path' => $file->getPathname(),
                ];
            })
            ->sortBy('name')
            ->values();
    }

    /**
     * List all available templates as array.
     */
    public function listTemplates(): array
    {
        return $this->list()->toArray();
    }

    /**
     * Preview a template with variable substitution.
     */
    public function previewTemplate(string $templateSlug, array $variables = []): ?array
    {
        $template = $this->get($templateSlug);

        if (! $template) {
            return null;
        }

        // Apply variable substitution
        $template = $this->substituteVariables($template, $variables);

        // Build preview structure
        return [
            'slug' => $templateSlug,
            'name' => $template['name'] ?? $templateSlug,
            'description' => $template['description'] ?? null,
            'category' => $template['category'] ?? 'general',
            'context' => $this->buildContext($template, $variables),
            'phases' => collect($template['phases'] ?? [])->map(function ($phase, $order) {
                return [
                    'order' => $order + 1,
                    'name' => $phase['name'] ?? 'Phase '.($order + 1),
                    'description' => $phase['description'] ?? null,
                    'tasks' => collect($phase['tasks'] ?? [])->map(function ($task) {
                        return is_string($task) ? ['name' => $task] : $task;
                    })->toArray(),
                ];
            })->toArray(),
            'variables_applied' => $variables,
            'guidelines' => $template['guidelines'] ?? [],
        ];
    }

    /**
     * Get a specific template by slug.
     */
    public function get(string $slug): ?array
    {
        $path = $this->templatesPath.'/'.$slug.'.yaml';

        if (! File::exists($path)) {
            $path = $this->templatesPath.'/'.$slug.'.yml';
        }

        if (! File::exists($path)) {
            return null;
        }

        $content = Yaml::parseFile($path);
        $content['slug'] = $slug;

        return $content;
    }

    /**
     * Create a plan from a template.
     */
    public function createPlan(
        string $templateSlug,
        array $variables = [],
        array $options = [],
        ?Workspace $workspace = null
    ): ?AgentPlan {
        $template = $this->get($templateSlug);

        if (! $template) {
            return null;
        }

        // Snapshot the raw template content before variable substitution so the
        // version record captures the canonical template, not the instantiated copy.
        $templateVersion = PlanTemplateVersion::findOrCreateFromTemplate($templateSlug, $template);

        // Replace variables in template
        $template = $this->substituteVariables($template, $variables);

        // Generate plan title and slug
        $title = $options['title'] ?? $template['name'];
        $planSlug = $options['slug'] ?? AgentPlan::generateSlug($title);

        // Build context from template
        $context = $this->buildContext($template, $variables);

        // Create the plan
        $plan = AgentPlan::create([
            'workspace_id' => $workspace?->id ?? $options['workspace_id'] ?? null,
            'slug' => $planSlug,
            'title' => $title,
            'description' => $template['description'] ?? null,
            'context' => $context,
            'status' => ($options['activate'] ?? false) ? AgentPlan::STATUS_ACTIVE : AgentPlan::STATUS_DRAFT,
            'template_version_id' => $templateVersion->id,
            'metadata' => array_merge($template['metadata'] ?? [], [
                'source' => 'template',
                'template_slug' => $templateSlug,
                'template_name' => $template['name'],
                'template_version' => $templateVersion->version,
                'variables' => $variables,
                'created_at' => now()->toIso8601String(),
            ]),
        ]);

        // Create phases
        foreach ($template['phases'] ?? [] as $order => $phaseData) {
            $tasks = [];
            foreach ($phaseData['tasks'] ?? [] as $task) {
                $tasks[] = is_string($task)
                    ? ['name' => $task, 'status' => 'pending']
                    : array_merge(['status' => 'pending'], $task);
            }

            AgentPhase::create([
                'agent_plan_id' => $plan->id,
                'order' => $order + 1,
                'name' => $phaseData['name'] ?? 'Phase '.($order + 1),
                'description' => $phaseData['description'] ?? null,
                'tasks' => $tasks,
                'dependencies' => $phaseData['dependencies'] ?? null,
                'metadata' => $phaseData['metadata'] ?? null,
            ]);
        }

        return $plan->fresh(['agentPhases']);
    }

    /**
     * Extract variable placeholders from template.
     */
    protected function extractVariables(array $template): array
    {
        $json = json_encode($template);
        preg_match_all('/\{\{\s*(\w+)\s*\}\}/', $json, $matches);

        $variables = array_unique($matches[1] ?? []);

        // Check for variable definitions in template
        $definitions = $template['variables'] ?? [];

        return collect($variables)->map(function ($var) use ($definitions) {
            $def = $definitions[$var] ?? [];

            return [
                'name' => $var,
                'description' => $def['description'] ?? null,
                'default' => $def['default'] ?? null,
                'required' => $def['required'] ?? true,
            ];
        })->values()->toArray();
    }

    /**
     * Substitute variables in template content.
     *
     * Uses a safe replacement strategy that properly escapes values for JSON context
     * to prevent corruption from special characters.
     */
    protected function substituteVariables(array $template, array $variables): array
    {
        $json = json_encode($template, JSON_UNESCAPED_UNICODE);

        foreach ($variables as $key => $value) {
            // Sanitise value: only allow scalar values
            if (! is_scalar($value) && $value !== null) {
                continue;
            }

            // Escape the value for safe JSON string insertion
            // json_encode wraps in quotes, so we extract just the escaped content
            $escapedValue = $this->escapeForJson((string) $value);

            $json = preg_replace(
                '/\{\{\s*'.preg_quote($key, '/').'\s*\}\}/',
                $escapedValue,
                $json
            );
        }

        // Apply defaults for unsubstituted variables
        foreach ($template['variables'] ?? [] as $key => $def) {
            if (isset($def['default']) && ! isset($variables[$key])) {
                $escapedDefault = $this->escapeForJson((string) $def['default']);
                $json = preg_replace(
                    '/\{\{\s*'.preg_quote($key, '/').'\s*\}\}/',
                    $escapedDefault,
                    $json
                );
            }
        }

        $result = json_decode($json, true);

        // Validate JSON decode was successful
        if ($result === null && json_last_error() !== JSON_ERROR_NONE) {
            // Return original template if substitution corrupted the JSON
            return $template;
        }

        return $result;
    }

    /**
     * Escape a string value for safe insertion into a JSON string context.
     *
     * This handles special characters that would break JSON structure:
     * - Backslashes, quotes, control characters
     */
    protected function escapeForJson(string $value): string
    {
        // json_encode the value, then strip the surrounding quotes
        $encoded = json_encode($value, JSON_UNESCAPED_UNICODE);

        // Handle encoding failure
        if ($encoded === false) {
            return '';
        }

        // Remove surrounding quotes from json_encode output
        return substr($encoded, 1, -1);
    }

    /**
     * Build context string from template.
     */
    protected function buildContext(array $template, array $variables): ?string
    {
        $context = $template['context'] ?? null;

        if (! $context) {
            // Build default context
            $lines = [];
            $lines[] = "## Plan: {$template['name']}";

            if ($template['description'] ?? null) {
                $lines[] = "\n{$template['description']}";
            }

            if (! empty($variables)) {
                $lines[] = "\n### Variables";
                foreach ($variables as $key => $value) {
                    $lines[] = "- **{$key}**: {$value}";
                }
            }

            if ($template['guidelines'] ?? null) {
                $lines[] = "\n### Guidelines";
                foreach ((array) $template['guidelines'] as $guideline) {
                    $lines[] = "- {$guideline}";
                }
            }

            $context = implode("\n", $lines);
        }

        return $context;
    }

    /**
     * Validate variables against template requirements.
     *
     * Returns a result array with:
     *   - valid: bool
     *   - errors: string[] – actionable messages including description and examples
     *   - naming_convention: string – reminder that variable names use snake_case
     */
    public function validateVariables(string $templateSlug, array $variables): array
    {
        $template = $this->get($templateSlug);

        if (! $template) {
            return ['valid' => false, 'errors' => ['Template not found'], 'naming_convention' => self::NAMING_CONVENTION];
        }

        $errors = [];

        foreach ($template['variables'] ?? [] as $name => $varDef) {
            $required = $varDef['required'] ?? true;

            if ($required && ! isset($variables[$name]) && ! isset($varDef['default'])) {
                $errors[] = $this->buildVariableError($name, $varDef);
            }
        }

        return [
            'valid' => empty($errors),
            'errors' => $errors,
            'naming_convention' => self::NAMING_CONVENTION,
        ];
    }

    /**
<<<<<<< HEAD
     * Naming convention reminder included in validation results.
     */
    private const NAMING_CONVENTION = 'Variable names use snake_case (e.g. project_name, api_key)';

    /**
     * Build an actionable error message for a missing required variable.
     *
     * Incorporates the variable's description, example values, and expected
     * format so the caller knows exactly what to provide.
     */
    private function buildVariableError(string $name, array $varDef): string
    {
        $message = "Required variable '{$name}' is missing";

        if (! empty($varDef['description'])) {
            $message .= ": {$varDef['description']}";
        }

        $hints = [];

        if (! empty($varDef['format'])) {
            $hints[] = "expected format: {$varDef['format']}";
        }

        if (! empty($varDef['example'])) {
            $hints[] = "example: '{$varDef['example']}'";
        } elseif (! empty($varDef['examples'])) {
            $exampleValues = is_array($varDef['examples'])
                ? array_slice($varDef['examples'], 0, 2)
                : [$varDef['examples']];
            $hints[] = "examples: '".implode("', '", $exampleValues)."'";
        }

        if (! empty($hints)) {
            $message .= ' ('.implode('; ', $hints).')';
        }

        return $message;
    }

    /**
     * Get the version history for a template slug, newest first.
     *
     * Returns an array of version summaries (without full content) for display.
     *
     * @return array<int, array{id: int, slug: string, version: int, name: string, content_hash: string, created_at: string}>
     */
    public function getVersionHistory(string $slug): array
    {
        return PlanTemplateVersion::historyFor($slug)
            ->map(fn (PlanTemplateVersion $v) => [
                'id' => $v->id,
                'slug' => $v->slug,
                'version' => $v->version,
                'name' => $v->name,
                'content_hash' => $v->content_hash,
                'created_at' => $v->created_at?->toIso8601String(),
            ])
            ->toArray();
    }

    /**
     * Get a specific stored version of a template by slug and version number.
     *
     * Returns the snapshotted content array, or null if not found.
     */
    public function getVersion(string $slug, int $version): ?array
    {
        $record = PlanTemplateVersion::where('slug', $slug)
            ->where('version', $version)
            ->first();

        return $record?->content;
    }

    /**
     * Get templates by category.
     */
    public function getByCategory(string $category): Collection
    {
        return $this->list()->filter(fn ($t) => $t['category'] === $category);
    }

    /**
     * Get template categories.
     */
    public function getCategories(): Collection
    {
        return $this->list()
            ->pluck('category')
            ->unique()
            ->sort()
            ->values();
    }
}
