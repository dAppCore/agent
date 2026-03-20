<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Core\Tenant\Models\Workspace;
use Illuminate\Contracts\View\View;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithPagination;
use Symfony\Component\HttpFoundation\StreamedResponse;

#[Title('API Keys')]
#[Layout('hub::admin.layouts.app')]
class ApiKeys extends Component
{
    use WithPagination;

    #[Url]
    public string $workspace = '';

    #[Url]
    public string $status = '';

    public int $perPage = 25;

    // Create modal
    public bool $showCreateModal = false;

    public string $newKeyName = '';

    public int $newKeyWorkspace = 0;

    public array $newKeyPermissions = [];

    public int $newKeyRateLimit = 100;

    public string $newKeyExpiry = '';

    // Created key display
    public bool $showCreatedKeyModal = false;

    public ?string $createdPlainKey = null;

    // Edit modal
    public bool $showEditModal = false;

    public ?int $editingKeyId = null;

    public array $editingPermissions = [];

    public int $editingRateLimit = 100;

    // IP restriction fields for create
    public bool $newKeyIpRestrictionEnabled = false;

    public string $newKeyIpWhitelist = '';

    // IP restriction fields for edit
    public bool $editingIpRestrictionEnabled = false;

    public string $editingIpWhitelist = '';

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function keys(): \Illuminate\Contracts\Pagination\LengthAwarePaginator
    {
        $query = AgentApiKey::with('workspace')
            ->orderByDesc('created_at');

        if ($this->workspace) {
            $query->where('workspace_id', $this->workspace);
        }

        if ($this->status === 'active') {
            $query->active();
        } elseif ($this->status === 'revoked') {
            $query->revoked();
        } elseif ($this->status === 'expired') {
            $query->expired();
        }

        return $query->paginate($this->perPage);
    }

    #[Computed]
    public function workspaces(): Collection
    {
        return Workspace::orderBy('name')->get();
    }

    #[Computed]
    public function availablePermissions(): array
    {
        return AgentApiKey::availablePermissions();
    }

    #[Computed]
    public function stats(): array
    {
        $baseQuery = AgentApiKey::query();

        if ($this->workspace) {
            $baseQuery->where('workspace_id', $this->workspace);
        }

        $total = (clone $baseQuery)->count();
        $active = (clone $baseQuery)->active()->count();
        $revoked = (clone $baseQuery)->revoked()->count();
        $totalCalls = (clone $baseQuery)->sum('call_count');

        return [
            'total' => $total,
            'active' => $active,
            'revoked' => $revoked,
            'total_calls' => $totalCalls,
        ];
    }

    #[Computed]
    public function editingKey(): ?AgentApiKey
    {
        if (! $this->editingKeyId) {
            return null;
        }

        return AgentApiKey::find($this->editingKeyId);
    }

    #[Computed]
    public function currentUserIp(): string
    {
        return request()->ip() ?? '127.0.0.1';
    }

    public function openCreateModal(): void
    {
        $this->newKeyName = '';
        $this->newKeyWorkspace = $this->workspaces->first()?->id ?? 0;
        $this->newKeyPermissions = [];
        $this->newKeyRateLimit = 100;
        $this->newKeyExpiry = '';
        $this->newKeyIpRestrictionEnabled = false;
        $this->newKeyIpWhitelist = '';
        $this->showCreateModal = true;
    }

    public function closeCreateModal(): void
    {
        $this->showCreateModal = false;
        $this->resetValidation();
    }

    public function createKey(): void
    {
        $rules = [
            'newKeyName' => 'required|string|max:255',
            'newKeyWorkspace' => 'required|exists:workspaces,id',
            'newKeyPermissions' => 'required|array|min:1',
            'newKeyRateLimit' => 'required|integer|min:1|max:10000',
        ];

        $messages = [
            'newKeyPermissions.required' => 'Select at least one permission.',
            'newKeyPermissions.min' => 'Select at least one permission.',
        ];

        // Add IP whitelist validation if enabled
        if ($this->newKeyIpRestrictionEnabled && empty(trim($this->newKeyIpWhitelist))) {
            $this->addError('newKeyIpWhitelist', 'IP whitelist is required when restrictions are enabled.');

            return;
        }

        $this->validate($rules, $messages);

        // Parse IP whitelist if enabled
        $ipWhitelist = [];
        if ($this->newKeyIpRestrictionEnabled && ! empty($this->newKeyIpWhitelist)) {
            $service = app(AgentApiKeyService::class);
            $parsed = $service->parseIpWhitelistInput($this->newKeyIpWhitelist);

            if (! empty($parsed['errors'])) {
                $this->addError('newKeyIpWhitelist', 'Invalid entries: '.implode(', ', $parsed['errors']));

                return;
            }

            $ipWhitelist = $parsed['entries'];
        }

        $expiresAt = null;
        if ($this->newKeyExpiry) {
            $expiresAt = match ($this->newKeyExpiry) {
                '30days' => now()->addDays(30),
                '90days' => now()->addDays(90),
                '1year' => now()->addYear(),
                default => null,
            };
        }

        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $this->newKeyWorkspace,
            $this->newKeyName,
            $this->newKeyPermissions,
            $this->newKeyRateLimit,
            $expiresAt
        );

        // Update IP restrictions if enabled
        if ($this->newKeyIpRestrictionEnabled) {
            $service->updateIpRestrictions($key, true, $ipWhitelist);
        }

        // Store the plaintext key for display
        $this->createdPlainKey = $key->plainTextKey;

        $this->showCreateModal = false;
        $this->showCreatedKeyModal = true;
    }

    public function closeCreatedKeyModal(): void
    {
        $this->showCreatedKeyModal = false;
        $this->createdPlainKey = null;
    }

    public function openEditModal(int $keyId): void
    {
        $key = AgentApiKey::find($keyId);
        if (! $key) {
            return;
        }

        $this->editingKeyId = $keyId;
        $this->editingPermissions = $key->permissions ?? [];
        $this->editingRateLimit = $key->rate_limit;
        $this->editingIpRestrictionEnabled = $key->ip_restriction_enabled ?? false;
        $this->editingIpWhitelist = implode("\n", $key->ip_whitelist ?? []);
        $this->showEditModal = true;
    }

    public function closeEditModal(): void
    {
        $this->showEditModal = false;
        $this->editingKeyId = null;
        $this->resetValidation();
    }

    public function updateKey(): void
    {
        $this->validate([
            'editingPermissions' => 'required|array|min:1',
            'editingRateLimit' => 'required|integer|min:1|max:10000',
        ]);

        // Validate IP whitelist if enabled
        if ($this->editingIpRestrictionEnabled && empty(trim($this->editingIpWhitelist))) {
            $this->addError('editingIpWhitelist', 'IP whitelist is required when restrictions are enabled.');

            return;
        }

        $key = AgentApiKey::find($this->editingKeyId);
        if (! $key) {
            return;
        }

        $service = app(AgentApiKeyService::class);
        $service->updatePermissions($key, $this->editingPermissions);
        $service->updateRateLimit($key, $this->editingRateLimit);

        // Parse and update IP restrictions
        $ipWhitelist = [];
        if ($this->editingIpRestrictionEnabled && ! empty($this->editingIpWhitelist)) {
            $parsed = $service->parseIpWhitelistInput($this->editingIpWhitelist);

            if (! empty($parsed['errors'])) {
                $this->addError('editingIpWhitelist', 'Invalid entries: '.implode(', ', $parsed['errors']));

                return;
            }

            $ipWhitelist = $parsed['entries'];
        }

        $service->updateIpRestrictions($key, $this->editingIpRestrictionEnabled, $ipWhitelist);

        $this->closeEditModal();
    }

    public function revokeKey(int $keyId): void
    {
        $key = AgentApiKey::find($keyId);
        if (! $key) {
            return;
        }

        $service = app(AgentApiKeyService::class);
        $service->revoke($key);
    }

    public function clearFilters(): void
    {
        $this->workspace = '';
        $this->status = '';
        $this->resetPage();
    }

    public function getStatusBadgeClass(AgentApiKey $key): string
    {
        if ($key->isRevoked()) {
            return 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300';
        }

        if ($key->isExpired()) {
            return 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300';
        }

        return 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300';
    }

    /**
     * Export API key usage data as CSV.
     */
    public function exportUsageCsv(): StreamedResponse
    {
        $filename = sprintf('api-key-usage-%s.csv', now()->format('Y-m-d'));

        return response()->streamDownload(function () {
            $handle = fopen('php://output', 'w');

            // Header
            fputcsv($handle, ['API Key Usage Export']);
            fputcsv($handle, ['Generated', now()->format('Y-m-d H:i:s')]);
            fputcsv($handle, []);

            // Summary stats
            fputcsv($handle, ['Summary Statistics']);
            fputcsv($handle, ['Metric', 'Value']);
            fputcsv($handle, ['Total Keys', $this->stats['total']]);
            fputcsv($handle, ['Active Keys', $this->stats['active']]);
            fputcsv($handle, ['Revoked Keys', $this->stats['revoked']]);
            fputcsv($handle, ['Total API Calls', $this->stats['total_calls']]);
            fputcsv($handle, []);

            // API Keys
            fputcsv($handle, ['API Keys']);
            fputcsv($handle, ['ID', 'Name', 'Workspace', 'Status', 'Permissions', 'Rate Limit', 'Call Count', 'Last Used', 'Expires', 'Created']);

            $query = AgentApiKey::with('workspace');

            if ($this->workspace) {
                $query->where('workspace_id', $this->workspace);
            }

            if ($this->status === 'active') {
                $query->active();
            } elseif ($this->status === 'revoked') {
                $query->revoked();
            } elseif ($this->status === 'expired') {
                $query->expired();
            }

            foreach ($query->orderByDesc('created_at')->cursor() as $key) {
                fputcsv($handle, [
                    $key->id,
                    $key->name,
                    $key->workspace?->name ?? 'N/A',
                    $key->getStatusLabel(),
                    implode(', ', $key->permissions ?? []),
                    $key->rate_limit.'/min',
                    $key->call_count,
                    $key->last_used_at?->format('Y-m-d H:i:s') ?? 'Never',
                    $key->expires_at?->format('Y-m-d H:i:s') ?? 'Never',
                    $key->created_at->format('Y-m-d H:i:s'),
                ]);
            }

            fclose($handle);
        }, $filename, [
            'Content-Type' => 'text/csv',
        ]);
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render(): View
    {
        return view('agentic::admin.api-keys');
    }
}
