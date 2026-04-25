<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Livewire;

use Core\Mod\Agentic\Actions\Brain\ForgetKnowledge;
use Core\Mod\Agentic\Actions\Brain\ListKnowledge;
use Core\Mod\Agentic\Actions\Brain\RecallKnowledge;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Tenant\Models\Workspace;
use Flux\Flux;
use Illuminate\Contracts\View\View;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Brain Explorer')]
#[Layout('hub::admin.layouts.app')]
class BrainExplorer extends Component
{
    public int $workspaceId = 0;

    public string $query = '';

    public string $typeFilter = '';

    public string $projectFilter = '';

    public string $agentFilter = '';

    public int $limit = 10;

    /**
     * @var array<int, array<string, mixed>>
     */
    public array $results = [];

    public bool $usedFallbackSearch = false;

    public function mount(?int $workspaceId = null): void
    {
        $this->checkHadesAccess();
        $this->workspaceId = $workspaceId ?? $this->resolveWorkspaceId();
        $this->loadRecentMemories();
    }

    #[Computed]
    public function memoryTypes(): array
    {
        return BrainMemory::VALID_TYPES;
    }

    #[Computed]
    public function availableAgents(): array
    {
        if ($this->workspaceId <= 0) {
            return [];
        }

        try {
            return BrainMemory::query()
                ->forWorkspace($this->workspaceId)
                ->active()
                ->latestVersions()
                ->orderBy('agent_id')
                ->pluck('agent_id')
                ->filter(static fn (mixed $agentId): bool => is_string($agentId) && $agentId !== '')
                ->unique()
                ->values()
                ->all();
        } catch (\Throwable) {
            return [];
        }
    }

    public function searchMemories(): void
    {
        $this->validate([
            'workspaceId' => 'required|integer|min:1',
            'query' => 'nullable|string|max:2000',
            'typeFilter' => 'nullable|string|max:255',
            'projectFilter' => 'nullable|string|max:255',
            'agentFilter' => 'nullable|string|max:255',
            'limit' => 'required|integer|min:1|max:20',
        ]);

        if (trim($this->query) === '') {
            $this->loadRecentMemories();

            return;
        }

        try {
            $result = RecallKnowledge::run(
                trim($this->query),
                $this->workspaceId,
                $this->brainFilters(),
                $this->limit,
            );

            $this->results = collect($result['memories'] ?? [])
                ->map(fn (mixed $memory): array => $this->normaliseMemory($memory))
                ->values()
                ->all();
            $this->usedFallbackSearch = false;
        } catch (\Throwable) {
            $this->results = $this->fallbackSearch();
            $this->usedFallbackSearch = true;
        }
    }

    public function forgetMemory(string $memoryId): void
    {
        ForgetKnowledge::run(
            $memoryId,
            $this->workspaceId,
            (string) (auth()->id() ?? 'hades-ui'),
            $this->query !== '' ? "Forgotten from search: {$this->query}" : 'Forgotten from explorer',
        );

        $this->searchMemories();
        $this->toast('Memory Forgotten', 'The memory was removed from the brain index.', 'warning');
    }

    public function refreshExplorer(): void
    {
        $this->searchMemories();
        $this->dispatch('notify', message: 'Brain explorer refreshed');
    }

    public function typeBadgeVariant(string $type): string
    {
        return match ($type) {
            'decision', 'architecture' => 'warning',
            'fact', 'documentation', 'context' => 'success',
            'bug' => 'danger',
            default => 'zinc',
        };
    }

    public function render(): View
    {
        return view()->file($this->viewPath());
    }

    private function loadRecentMemories(): void
    {
        if ($this->workspaceId <= 0) {
            $this->results = [];
            $this->usedFallbackSearch = false;

            return;
        }

        try {
            $result = ListKnowledge::run($this->workspaceId, $this->brainFilters() + [
                'limit' => $this->limit,
            ]);

            $this->results = collect($result['memories'] ?? [])
                ->map(fn (mixed $memory): array => $this->normaliseMemory($memory))
                ->values()
                ->all();
            $this->usedFallbackSearch = false;
        } catch (\Throwable) {
            $this->results = [];
            $this->usedFallbackSearch = false;
        }
    }

    /**
     * @return array<string, mixed>
     */
    private function brainFilters(): array
    {
        return array_filter([
            'type' => $this->typeFilter !== '' ? $this->typeFilter : null,
            'project' => $this->projectFilter !== '' ? $this->projectFilter : null,
            'agent_id' => $this->agentFilter !== '' ? $this->agentFilter : null,
        ], static fn (mixed $value): bool => $value !== null && $value !== '');
    }

    /**
     * @return array<int, array<string, mixed>>
     */
    private function fallbackSearch(): array
    {
        $query = BrainMemory::query()
            ->forWorkspace($this->workspaceId)
            ->active()
            ->latestVersions()
            ->forProject($this->projectFilter !== '' ? $this->projectFilter : null)
            ->byAgent($this->agentFilter !== '' ? $this->agentFilter : null);

        if ($this->typeFilter !== '') {
            $query->ofType($this->typeFilter);
        }

        if (trim($this->query) !== '') {
            $like = '%'.trim($this->query).'%';

            $query->where(function ($builder) use ($like): void {
                $builder->where('content', 'like', $like)
                    ->orWhere('source', 'like', $like)
                    ->orWhere('project', 'like', $like)
                    ->orWhere('org', 'like', $like);
            });
        }

        return $query->orderByDesc('created_at')
            ->limit($this->limit)
            ->get()
            ->map(fn (BrainMemory $memory): array => $this->normaliseMemory($memory->toMcpContext()))
            ->values()
            ->all();
    }

    /**
     * @param  array<string, mixed>|BrainMemory  $memory
     * @return array<string, mixed>
     */
    private function normaliseMemory(array|BrainMemory $memory): array
    {
        if ($memory instanceof BrainMemory) {
            $memory = $memory->toMcpContext();
        }

        return [
            'id' => (string) ($memory['id'] ?? ''),
            'agent_id' => (string) ($memory['agent_id'] ?? 'unknown'),
            'type' => (string) ($memory['type'] ?? 'context'),
            'content' => (string) ($memory['content'] ?? ''),
            'tags' => array_values(array_filter($memory['tags'] ?? [], static fn (mixed $tag): bool => is_string($tag) && $tag !== '')),
            'project' => $memory['project'] ?? null,
            'org' => $memory['org'] ?? null,
            'confidence' => isset($memory['confidence']) ? (float) $memory['confidence'] : 0.0,
            'score' => isset($memory['score']) ? (float) $memory['score'] : null,
            'source' => $memory['source'] ?? null,
            'created_at' => $memory['created_at'] ?? null,
        ];
    }

    private function resolveWorkspaceId(): int
    {
        try {
            return (int) (Workspace::query()->value('id') ?? 0);
        } catch (\Throwable) {
            return 0;
        }
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    private function toast(string $heading, string $text, string $variant): void
    {
        if (class_exists(Flux::class)) {
            Flux::toast(
                heading: $heading,
                text: $text,
                variant: $variant,
            );

            return;
        }

        $this->dispatch('notify', message: $text, variant: $variant);
    }

    private function viewPath(): string
    {
        return __DIR__.'/../../resources/views/livewire/agentic/brain-explorer.blade.php';
    }
}
