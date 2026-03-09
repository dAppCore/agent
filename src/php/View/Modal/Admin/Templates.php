<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Services\PlanTemplateService;
use Core\Tenant\Models\Workspace;
use Flux\Flux;
use Illuminate\Contracts\View\View;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Str;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithFileUploads;
use Symfony\Component\Yaml\Exception\ParseException;
use Symfony\Component\Yaml\Yaml;

#[Title('Plan Templates')]
#[Layout('hub::admin.layouts.app')]
class Templates extends Component
{
    use WithFileUploads;

    #[Url]
    public string $category = '';

    #[Url]
    public string $search = '';

    // Preview modal
    public bool $showPreviewModal = false;

    public ?string $previewSlug = null;

    // Create plan modal
    public bool $showCreateModal = false;

    public ?string $createTemplateSlug = null;

    public int $createWorkspaceId = 0;

    public string $createTitle = '';

    public bool $createActivate = false;

    public array $createVariables = [];

    // Import modal
    public bool $showImportModal = false;

    public ?UploadedFile $importFile = null;

    public string $importFileName = '';

    public ?array $importPreview = null;

    public ?string $importError = null;

    protected PlanTemplateService $templateService;

    public function boot(PlanTemplateService $templateService): void
    {
        $this->templateService = $templateService;
    }

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function templates(): Collection
    {
        $templates = $this->templateService->list();

        if ($this->category) {
            $templates = $templates->filter(fn ($t) => $t['category'] === $this->category);
        }

        if ($this->search) {
            $search = strtolower($this->search);
            $templates = $templates->filter(fn ($t) => str_contains(strtolower($t['name']), $search)
                || str_contains(strtolower($t['description'] ?? ''), $search)
            );
        }

        return $templates->values();
    }

    #[Computed]
    public function categories(): Collection
    {
        return $this->templateService->getCategories();
    }

    #[Computed]
    public function workspaces(): Collection
    {
        return Workspace::orderBy('name')->get();
    }

    #[Computed]
    public function previewTemplate(): ?array
    {
        if (! $this->previewSlug) {
            return null;
        }

        return $this->templateService->previewTemplate($this->previewSlug, []);
    }

    #[Computed]
    public function createTemplate(): ?array
    {
        if (! $this->createTemplateSlug) {
            return null;
        }

        return $this->templateService->get($this->createTemplateSlug);
    }

    #[Computed]
    public function createPreview(): ?array
    {
        if (! $this->createTemplateSlug) {
            return null;
        }

        return $this->templateService->previewTemplate($this->createTemplateSlug, $this->createVariables);
    }

    #[Computed]
    public function stats(): array
    {
        $templates = $this->templateService->list();

        return [
            'total' => $templates->count(),
            'categories' => $templates->pluck('category')->unique()->count(),
            'total_phases' => $templates->sum('phases_count'),
            'with_variables' => $templates->filter(fn ($t) => count($t['variables'] ?? []) > 0)->count(),
        ];
    }

    public function openPreview(string $slug): void
    {
        $this->previewSlug = $slug;
        $this->showPreviewModal = true;
    }

    public function closePreview(): void
    {
        $this->showPreviewModal = false;
        $this->previewSlug = null;
    }

    public function openCreateModal(string $slug): void
    {
        $template = $this->templateService->get($slug);

        if (! $template) {
            Flux::toast(
                heading: 'Template Not Found',
                text: 'The selected template could not be loaded.',
                variant: 'danger',
            );

            return;
        }

        $this->createTemplateSlug = $slug;
        $this->createTitle = $template['name'];
        $this->createWorkspaceId = $this->workspaces->first()?->id ?? 0;
        $this->createActivate = false;

        // Initialise variables with defaults
        $this->createVariables = [];
        foreach ($template['variables'] ?? [] as $name => $config) {
            $this->createVariables[$name] = $config['default'] ?? '';
        }

        $this->showCreateModal = true;
    }

    public function closeCreateModal(): void
    {
        $this->showCreateModal = false;
        $this->createTemplateSlug = null;
        $this->createVariables = [];
        $this->resetValidation();
    }

    public function createPlan(): void
    {
        // Validate required variables
        $template = $this->templateService->get($this->createTemplateSlug);

        if (! $template) {
            Flux::toast(
                heading: 'Template Not Found',
                text: 'The selected template could not be loaded.',
                variant: 'danger',
            );

            return;
        }

        $rules = [
            'createWorkspaceId' => 'required|exists:workspaces,id',
            'createTitle' => 'required|string|max:255',
        ];

        // Add variable validation
        foreach ($template['variables'] ?? [] as $name => $config) {
            if ($config['required'] ?? false) {
                $rules["createVariables.{$name}"] = 'required|string';
            }
        }

        $this->validate($rules, [
            'createVariables.*.required' => 'This variable is required.',
        ]);

        // Validate variables using service
        $validation = $this->templateService->validateVariables($this->createTemplateSlug, $this->createVariables);

        if (! $validation['valid']) {
            foreach ($validation['errors'] as $error) {
                $this->addError('createVariables', $error);
            }

            return;
        }

        // Create the plan
        $workspace = Workspace::find($this->createWorkspaceId);
        $plan = $this->templateService->createPlan(
            $this->createTemplateSlug,
            $this->createVariables,
            [
                'title' => $this->createTitle,
                'activate' => $this->createActivate,
            ],
            $workspace
        );

        if (! $plan) {
            Flux::toast(
                heading: 'Creation Failed',
                text: 'Failed to create plan from template.',
                variant: 'danger',
            );

            return;
        }

        $this->closeCreateModal();

        Flux::toast(
            heading: 'Plan Created',
            text: "Plan '{$plan->title}' has been created from template.",
            variant: 'success',
        );

        // Redirect to the new plan
        $this->redirect(route('hub.agents.plans.show', $plan->slug), navigate: true);
    }

    public function openImportModal(): void
    {
        $this->importFile = null;
        $this->importFileName = '';
        $this->importPreview = null;
        $this->importError = null;
        $this->showImportModal = true;
    }

    public function closeImportModal(): void
    {
        $this->showImportModal = false;
        $this->importFile = null;
        $this->importFileName = '';
        $this->importPreview = null;
        $this->importError = null;
        $this->resetValidation();
    }

    public function updatedImportFile(): void
    {
        $this->importError = null;
        $this->importPreview = null;

        if (! $this->importFile) {
            return;
        }

        try {
            $content = file_get_contents($this->importFile->getRealPath());
            $parsed = Yaml::parse($content);

            // Validate basic structure
            if (! is_array($parsed)) {
                $this->importError = 'Invalid YAML format: expected an object.';

                return;
            }

            if (! isset($parsed['name'])) {
                $this->importError = 'Template must have a "name" field.';

                return;
            }

            if (! isset($parsed['phases']) || ! is_array($parsed['phases'])) {
                $this->importError = 'Template must have a "phases" array.';

                return;
            }

            // Generate slug from filename
            $originalName = $this->importFile->getClientOriginalName();
            $slug = Str::slug(pathinfo($originalName, PATHINFO_FILENAME));

            // Check for duplicate slug
            $existingPath = resource_path("plan-templates/{$slug}.yaml");
            if (File::exists($existingPath)) {
                $slug = $slug.'-'.Str::random(4);
            }

            $this->importFileName = $slug;

            // Build preview
            $this->importPreview = [
                'name' => $parsed['name'],
                'description' => $parsed['description'] ?? null,
                'category' => $parsed['category'] ?? 'custom',
                'phases_count' => count($parsed['phases']),
                'variables_count' => count($parsed['variables'] ?? []),
                'has_guidelines' => isset($parsed['guidelines']) && count($parsed['guidelines']) > 0,
            ];
        } catch (ParseException $e) {
            $this->importError = 'Invalid YAML syntax: '.$e->getMessage();
        } catch (\Exception $e) {
            $this->importError = 'Error reading file: '.$e->getMessage();
        }
    }

    public function importTemplate(): void
    {
        if (! $this->importFile || ! $this->importPreview) {
            $this->importError = 'Please select a valid YAML file.';

            return;
        }

        $this->validate([
            'importFileName' => 'required|string|regex:/^[a-z0-9-]+$/|max:64',
        ], [
            'importFileName.regex' => 'Filename must contain only lowercase letters, numbers, and hyphens.',
        ]);

        try {
            $content = file_get_contents($this->importFile->getRealPath());
            $targetPath = resource_path("plan-templates/{$this->importFileName}.yaml");

            // Check for existing file
            if (File::exists($targetPath)) {
                $this->importError = 'A template with this filename already exists.';

                return;
            }

            // Ensure directory exists
            $dir = resource_path('plan-templates');
            if (! File::isDirectory($dir)) {
                File::makeDirectory($dir, 0755, true);
            }

            // Save the file
            File::put($targetPath, $content);

            $this->closeImportModal();

            Flux::toast(
                heading: 'Template Imported',
                text: "Template '{$this->importPreview['name']}' has been imported successfully.",
                variant: 'success',
            );
        } catch (\Exception $e) {
            $this->importError = 'Failed to save template: '.$e->getMessage();
        }
    }

    public function deleteTemplate(string $slug): void
    {
        $path = resource_path("plan-templates/{$slug}.yaml");

        if (! File::exists($path)) {
            $path = resource_path("plan-templates/{$slug}.yml");
        }

        if (! File::exists($path)) {
            Flux::toast(
                heading: 'Template Not Found',
                text: 'The template file could not be found.',
                variant: 'danger',
            );

            return;
        }

        // Get template name for toast
        $template = $this->templateService->get($slug);
        $name = $template['name'] ?? $slug;

        File::delete($path);

        Flux::toast(
            heading: 'Template Deleted',
            text: "Template '{$name}' has been deleted.",
            variant: 'warning',
        );
    }

    public function clearFilters(): void
    {
        $this->category = '';
        $this->search = '';
    }

    public function getCategoryColor(string $category): string
    {
        return match ($category) {
            'development' => 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
            'maintenance' => 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300',
            'review' => 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
            'migration' => 'bg-purple-100 text-purple-700 dark:bg-purple-900/50 dark:text-purple-300',
            'custom' => 'bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
            default => 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300',
        };
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render(): View
    {
        return view('agentic::admin.templates');
    }
}
